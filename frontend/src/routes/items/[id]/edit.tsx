import { useLocation, type RouteProps } from "@solidjs/router";
import { Show } from "solid-js";
import controls from "../../../components/Controls.module.css";
import Page from "../../../components/Page";
import DraftEditor from "../../../features/editor/DraftEditor";

export default function Edit(props: RouteProps<"/items/:id/edit">) {
  const location = useLocation();
  const revision = () => new URLSearchParams(location.search).get("mode") === "revision";
  return (
    <Show when={revision()} fallback={<DraftEditor publishedId={props.params.id} />}>
      <Page
        title="Edit Text & Voice"
        description="Working revisions are unavailable in this version."
      >
        <a class={controls.secondary} href={`/items/${encodeURIComponent(props.params.id)}/edit`}>
          Back to Edit Article
        </a>
      </Page>
    </Show>
  );
}
