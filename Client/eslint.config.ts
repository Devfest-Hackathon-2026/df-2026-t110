import eslintPluginVue from 'eslint-plugin-vue'
import eslintConfigPrettier from 'eslint-config-prettier'
import { withVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

export default withVueTs(
  { ignores: ['dist/**', 'node_modules/**', 'public/vendor/**', 'src/api/schema.d.ts'] },
  eslintPluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  eslintConfigPrettier,
)
