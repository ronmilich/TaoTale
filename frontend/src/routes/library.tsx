import { Title } from "@solidjs/meta";
import { createMemo, createSignal } from "solid-js";
import controls from "../components/Controls.module.css";
import { samples } from "../features/library/samples";
import cover from "../features/library/hangzhou-tea.webp";
import { usePreferences } from "../preferences/Preferences";
import styles from "../features/library/Library.module.css";

export default function Library() {
  const preferences = usePreferences();
  const article = createMemo(() => samples[preferences.language()]);
  const [favorites, setFavorites] = createSignal<string[]>([]);
  const isFavorite = createMemo(() => favorites().includes(article().id));

  function toggleFavorite() {
    const id = article().id;
    setFavorites((previous) =>
      previous.includes(id) ? previous.filter((entry) => entry !== id) : [...previous, id],
    );
  }

  return (
    <>
      <Title>Library · TaoTale</Title>
      <section class={styles.intro}>
        <div>
          <h1>Your reading library</h1>
          <p>Keep articles to read and revisit in your learning language.</p>
        </div>
        <a class={controls.primary} href="/create">
          <span aria-hidden="true">＋</span> New article
        </a>
      </section>
      <section class={styles.library} aria-labelledby="sample-heading">
        <h2 id="sample-heading">Sample article</h2>
        <div class={styles.grid}>
          <article class={styles.card}>
            <div class={styles.cover}>
              <img src={cover} alt="" width="760" height="420" />
              <button
                type="button"
                class={controls.favorite}
                aria-label="Favorite sample"
                aria-pressed={isFavorite() ? "true" : "false"}
                onClick={toggleFavorite}
              >
                <span aria-hidden="true">{isFavorite() ? "★" : "☆"}</span>
              </button>
            </div>
            <div class={styles.details}>
              <h2>
                <a href={`/items/${article().id}`}>{article().title}</a>
              </h2>
              <p class={styles.excerpt} lang={article().language}>
                {article().excerpt}
              </p>
              <div class={styles.tags}>
                <span>Intermediate</span>
                <span>Culture</span>
              </div>
              <div class={styles.cardFooter}>
                <span>{article().language === "ko-KR" ? "Korean" : "Mandarin Chinese"}</span>
                <a class={controls.secondary} href={`/items/${article().id}`}>
                  Open reader <span aria-hidden="true">→</span>
                </a>
              </div>
            </div>
          </article>
        </div>
      </section>
    </>
  );
}
