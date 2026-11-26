import Page from "../components/Page";
import styles from "../components/Page.module.css";
import { usePreferences } from "../preferences/Preferences";
import { validatePreference } from "../preferences/storage";

export default function Settings() {
  const preferences = usePreferences();
  return (
    <Page title="Settings">
      <section class={styles.panel} aria-label="Appearance">
        <div class={styles.field}>
          <label for="theme">Appearance</label>
          <select
            id="theme"
            value={preferences.theme()}
            aria-describedby="theme-help"
            onChange={(event) =>
              preferences.changeTheme(validatePreference("theme", event.currentTarget.value))
            }
          >
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
          <p id="theme-help">System follows your device’s color setting.</p>
        </div>
      </section>
    </Page>
  );
}
