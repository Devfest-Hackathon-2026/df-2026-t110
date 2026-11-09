import eslintPluginVue from 'eslint-plugin-vue'
import eslintConfigPrettier from 'eslint-config-prettier'
import { withVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

export default withVueTs(
  { ignores: ['dist/**', 'node_modules/**', 'public/vendor/**', 'src/api/schema.d.ts'] },
  eslintPluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  {
    files: ['src/features/picolang/language/**/*.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              regex: '^(?!\\./|(?:\\.\\./){3}api/schema\\.d\\.ts$)',
              message:
                'language may import only language modules and generated API types; UI dependencies belong in editor.',
            },
          ],
        },
      ],
    },
  },
  eslintConfigPrettier,
)
