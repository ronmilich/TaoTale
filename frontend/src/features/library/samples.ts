import type { Language } from "../../preferences/storage";

export const samples = {
  "zh-CN": {
    id: "hangzhou-morning",
    language: "zh-CN",
    title: "A quiet morning in Hangzhou",
    excerpt: "天刚亮，茶馆里已经飘着茉莉花的香气。",
    text: "天刚亮，茶馆里已经飘着茉莉花的香气。\n\n窗外的花带着露水。我坐在窗边，听风吹过树叶，慢慢喝完了一杯茶。",
  },
  "ko-KR": {
    id: "hangzhou-morning-ko",
    language: "ko-KR",
    title: "A quiet morning in Hangzhou",
    excerpt: "해가 뜨자 찻집에는 벌써 재스민 향기가 퍼졌어요.",
    text: "해가 뜨자 찻집에는 벌써 재스민 향기가 퍼졌어요.\n\n창밖의 꽃에는 이슬이 맺혀 있었어요. 저는 창가에 앉아 바람에 나뭇잎이 흔들리는 소리를 들으며 차를 천천히 마셨어요.",
  },
} satisfies Record<
  Language,
  { id: string; language: Language; title: string; excerpt: string; text: string }
>;
