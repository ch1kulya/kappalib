export function initDescription(): void {
  const desc = document.getElementById("novel-description");
  if (!desc) return;

  const checkIfNeedsToggle = (): void => {
    const lineHeightStr = getComputedStyle(desc).lineHeight;
    const lineHeight = parseFloat(lineHeightStr) || 24;
    const maxLines = 6;
    const maxHeight = lineHeight * maxLines;

    const wasExpanded = desc.classList.contains("expanded");
    desc.classList.remove("short", "visible", "expanded");

    if (desc.scrollHeight <= maxHeight + 5) {
      desc.classList.add("short");
      desc.style.cursor = "";
    } else {
      desc.classList.add("visible");
      desc.style.cursor = "pointer";
      if (wasExpanded) {
        desc.classList.add("expanded");
      }
    }
  };

  checkIfNeedsToggle();

  desc.addEventListener("click", function (e: MouseEvent) {
    if (e.target === desc && desc.classList.contains("visible")) {
      desc.classList.toggle("expanded");
    }
  });

  if (typeof ResizeObserver !== "undefined") {
    const resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(() => {
        checkIfNeedsToggle();
      });
    });

    resizeObserver.observe(desc);

    window.addEventListener("beforeunload", () => {
      resizeObserver.disconnect();
    });
  }
}
