import { useNavigate } from "@solidjs/router";
import { onSettled } from "solid-js";

export default function Home() {
  const navigate = useNavigate();
  onSettled(() => navigate("/library", { replace: true }));
  return <a href="/library">Open the Library</a>;
}
