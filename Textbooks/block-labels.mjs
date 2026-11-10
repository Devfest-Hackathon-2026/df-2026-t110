// UI labels use these characters outside the elementary-school kanji list.
// Permit them only in exact quotations accompanied by their readings.
const labelsWithReadings =
  /「(?:制御」（せいぎょ）|距離を返す」（きょりをかえす）)/g;

export function isBlockLabelKanji(message, text) {
  if (message.ruleId !== "ja-grade-kanji") return false;
  const start = message.loc.start;
  const index =
    message.range?.[0] ??
    text
      .split("\n")
      .slice(0, start.line - 1)
      .reduce((n, line) => n + line.length + 1, 0) +
      start.column -
      1;
  return [...text.matchAll(labelsWithReadings)].some(
    (match) =>
      index > match.index && index < match.index + match[0].indexOf("」"),
  );
}
