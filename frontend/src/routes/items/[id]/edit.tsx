import { useLocation, type RouteProps } from "@solidjs/router";
import { Show } from "solid-js";
import DraftEditor from "../../../features/editor/DraftEditor";
import RevisionEditor from "../../../features/editor/RevisionEditor";

export default function Edit(props: RouteProps<"/items/:id/edit">) {
  const location = useLocation();
  const revision = () => new URLSearchParams(location.search).get("mode") === "revision";
  return (
    <Show when={revision()} fallback={<DraftEditor publishedId={props.params.id} />}>
      <Show when={props.params.id} keyed>
        {(id) => <RevisionEditor itemId={id} />}
      </Show>
    </Show>
  );
}
