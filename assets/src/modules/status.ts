interface StatusData {
  indicator: "none" | "minor" | "major" | "critical" | "maintenance" | string;
  description: string;
}

interface StatusResponse {
  status: StatusData;
}

export function initStatusBadge(): void {
  const widget = document.getElementById("status-widget");
  const dot = widget?.querySelector<HTMLElement>(".status-dot");
  const text = widget?.querySelector<HTMLElement>(".status-text");

  if (!widget || !dot || !text) return;

  console.info("Fetching system status...");

  fetch("/status")
    .then((res) => {
      if (!res.ok) throw new Error("Network response was not ok");
      return res.json() as Promise<StatusResponse>;
    })
    .then((data) => {
      const indicator = data.status.indicator;
      const description = data.status.description;

      console.info(`System status received: ${indicator}`);

      widget.classList.remove("operational", "degraded", "outage");
      text.textContent = description;

      switch (indicator) {
        case "none":
          widget.classList.add("operational");
          break;
        case "minor":
          widget.classList.add("degraded");
          break;
        case "major":
        case "critical":
          widget.classList.add("outage");
          break;
        case "maintenance":
          widget.classList.add("degraded");
          break;
        default:
          dot.style.backgroundColor = "var(--tertiary)";
      }
    })
    .catch((err) => {
      console.error("Failed to fetch system status", err);
      text.textContent = "Статус недоступен";
      dot.style.backgroundColor = "var(--tertiary)";
    });
}
