// OpenAPI の Tutorial は id / title / fieldId のみを定義している。
// 説明文とサンプル盤面は、教材 API に内容が追加されるまでのローカル原稿。
export const LOCAL_LESSONS = [
  {
    id: 'first-step',
    title: '前に進む',
    fieldId: 'first-step',
    goal: '右隣のゴールへ進む',
    map: ['SG'],
    lesson: '「前進」を1回。向いている方向に1マス進むことから始めよう。',
    guide: [
      'こんにちは。ぼくは案内役のロボット。緑の相棒をゴールまで届けよう。',
      '相棒は右を向いているよ。ゴールはすぐ隣。必要な命令はいくつかな？',
      '「前進」を1回で到着。命令と動きがつながったら、次の課題へ進もう。',
    ],
  },
  {
    id: 'more-steps',
    title: 'もっと前に進む',
    fieldId: 'more-steps',
    goal: '長い道をゴールまで進む',
    map: ['S..G'],
    lesson: '同じ命令を並べて、ゴールまでの距離を確かめよう。',
    guide: [
      '今度は道が少し長いね。相棒とゴールの間に何マスあるか見てみよう。',
      '1回の「前進」で動くのは1マス。進む回数を数えてみて。',
      '同じ動きが続くときは、あとで「繰り返し」を使うと短く書けるよ。',
    ],
  },
  {
    id: 'turn',
    title: '左右に回る',
    fieldId: 'turn',
    goal: '曲がった先のゴールへ向かう',
    map: ['..G.', 'S...'],
    lesson: '進むだけでは届かない。向きを変える場所を探そう。',
    guide: [
      'ゴールが正面にないね。まずは進む道と曲がる場所を見つけよう。',
      '回転すると、相棒のいるマスは変わらずに向きだけが変わるよ。',
      '右へ進んでから左を向けば、ゴールへ続く道が見えるはず。',
    ],
  },
  {
    id: 'break-rock',
    title: '岩を壊す',
    fieldId: 'break-rock',
    goal: '岩を越えてゴールへ進む',
    map: ['S..#..G'],
    lesson: '目のビームで正面の岩を壊す。岩の手前まで進んでから使おう。',
    guide: [
      '道の途中に岩があるよ。岩のマスにはそのまま進めない。',
      '岩のひとつ手前で「目のビーム」を使うと、正面の岩を壊せるよ。',
      'ビームを使う位置と、その後に進む回数を考えてみよう。',
    ],
  },
]

export function localLesson(id) {
  return LOCAL_LESSONS.find(lesson => lesson.id === id) ?? null
}

export function lessonField(lesson) {
  return { id: lesson.fieldId, title: lesson.title, map: lesson.map }
}
