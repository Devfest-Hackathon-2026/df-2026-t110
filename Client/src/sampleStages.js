// 1つのフィールド。盤面サイズと岩の個数候補は共通で、海・荷物・置き場も配置ごとに変わる。
export const SAMPLE_FIELD = {
  id: 'crystal-route',
  title: '結晶の道',
  template: ['S?????G', '???????'],
  rockCounts: [1, 2],
  candidates: [
    { id: 'layout-a', map: ['S..#..G', '._O.X..'] },
    { id: 'layout-b', map: ['S._#..G', '..O.X..'] },
    { id: 'layout-c', map: ['S..##.G', '.O._X..'] },
    { id: 'layout-d', map: ['S.#..#G', '..O_.X.'] },
  ],
}
