import { globSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import Ajv2020 from "ajv/dist/2020.js";
import { JSON_SCHEMA, load } from "js-yaml";
import { createLinter, loadTextlintrc } from "textlint";
import { isBlockLabelKanji } from "./block-labels.mjs";

const textbooksDir = new URL("./", import.meta.url);
const schemaDir = new URL("../Schema/", textbooksDir);
const tutorialUrl = new URL("tutorial/tutorial-detail.schema.yaml", schemaDir);
const cwd = fileURLToPath(textbooksDir);
const [filename, ...extraArgs] = process.argv.slice(2);
if (extraArgs.length > 0) {
  console.error("Usage: pnpm -C Textbooks lint [filename.yaml]");
  process.exit(1);
}

const pages = globSync("pages/**/*.{yaml,yml}", { cwd }).sort();
const targets =
  filename === undefined
    ? pages
    : pages.filter((path) => path === filename || path === `pages/${filename}`);
if (filename !== undefined && targets.length !== 1) {
  console.error(`Expected one YAML file under Textbooks/pages: ${filename}`);
  process.exit(1);
}

function readYaml(path) {
  return load(readFileSync(path, "utf8"), { schema: JSON_SCHEMA });
}

const ajv = new Ajv2020({
  allErrors: true,
  // hint-id.yaml already checks the UUID syntax with its pattern.
  formats: { uuid: true },
  loadSchema: async (uri) => readYaml(new URL(uri)),
});
const validate = await ajv.compileAsync({
  ...readYaml(tutorialUrl),
  $id: tutorialUrl.href,
});

const descriptor = await loadTextlintrc({
  configFilePath: fileURLToPath(new URL(".textlintrc.json", textbooksDir)),
});
const linter = createLinter({ descriptor });

for (const path of targets) {
  let page;
  try {
    page = readYaml(resolve(cwd, path));
  } catch (error) {
    console.error(`${path} [YAML] ${error.message}`);
    process.exitCode = 1;
    continue;
  }

  if (!validate(page)) {
    for (const error of validate.errors) {
      console.error(
        `${path} [schema] ${error.instancePath || "/"} ${error.message}`,
      );
    }
    process.exitCode = 1;
  }

  const prose = ["title", "description", "goal", "lesson"].map((field) => [
    field,
    page?.[field],
  ]);
  if (Array.isArray(page?.guide)) {
    prose.push(...page.guide.map((line, index) => [`guide[${index}]`, line]));
  }
  for (const [field, value] of prose) {
    if (typeof value !== "string") continue;
    const { messages } = await linter.lintText(value, `${path}.${field}.txt`);
    for (const message of messages) {
      if (isBlockLabelKanji(message, value)) continue;
      console.error(
        `${path} [textlint] ${field}:${message.loc.start.line}:${message.loc.start.column}-${message.loc.end.line}:${message.loc.end.column} ${message.message} (${message.ruleId})`,
      );
      process.exitCode = 1;
    }
  }
}
