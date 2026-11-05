import { initTooltips } from "./tooltip";

interface DailyPoint {
  date: string;
  total_seconds: number;
  active_users: number;
}

export function initAboutPage(): void {
  const container = document.getElementById("about-charts-container");
  const timePlot = document.getElementById("chart-reading-time");
  const usersPlot = document.getElementById("chart-active-users");

  if (!container || !timePlot || !usersPlot) {
    return;
  }

  let dailyData: DailyPoint[] = [];
  try {
    dailyData = JSON.parse(container.dataset.daily || "[]");
  } catch {
    dailyData = [];
  }

  if (!dailyData.length) {
    dailyData = generateSampleData(30);
  }

  let currentDays = 7;

  function render(): void {
    const slice = dailyData.slice(-currentDays);

    const timeHours = slice.map((p) => Math.round((p.total_seconds / 3600) * 10) / 10);
    const totalHours = Math.round(
      slice.reduce((acc, p) => acc + p.total_seconds, 0) / 3600,
    );

    const timeTotalEl = document.getElementById("chart-time-total");
    const timeSummaryEl = document.getElementById("chart-time-summary");
    if (timeTotalEl) timeTotalEl.textContent = `${totalHours} ч`;
    if (timeSummaryEl) {
      timeSummaryEl.textContent = `за последние ${slice.length} дн.`;
    }

    renderKappalibChart(timePlot!, slice, timeHours, "ч");

    const userValues = slice.map((p) => p.active_users);
    const avgUsers = Math.round(
      userValues.reduce((a, b) => a + b, 0) / (userValues.length || 1),
    );

    const usersTotalEl = document.getElementById("chart-users-total");
    const usersSummaryEl = document.getElementById("chart-users-summary");
    if (usersTotalEl) {
      usersTotalEl.textContent = `${userValues[userValues.length - 1] || 0}`;
    }
    if (usersSummaryEl) {
      usersSummaryEl.textContent = `в среднем ${avgUsers} в день`;
    }

    renderKappalibChart(usersPlot!, slice, userValues, "чит.");

    initTooltips([timePlot!, usersPlot!]);
  }

  const tabBtns = document.querySelectorAll<HTMLButtonElement>(
    ".about-tab-btn",
  );
  tabBtns.forEach((btn) => {
    btn.onclick = () => {
      tabBtns.forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      currentDays = parseInt(btn.dataset.period || "7", 10);
      render();
    };
  });

  render();
}

function calculateStep(range: number): number {
  const raw = Math.max(range, 1) / 3;
  const magnitude = 10 ** Math.floor(Math.log10(raw || 1));
  const factor = raw <= magnitude
    ? 1
    : raw <= 2 * magnitude
    ? 2
    : raw <= 5 * magnitude
    ? 5
    : 10;
  return Math.max(1, factor * magnitude);
}

function renderKappalibChart(
  container: HTMLElement,
  data: DailyPoint[],
  values: number[],
  unit: string,
): void {
  const max = Math.max(0, ...values);
  const min = 0;
  const step = calculateStep(max - min);
  const top = Math.ceil(max / step) * step || step;
  const bottom = 0;
  const count = values.length;

  const position = (value: number) => (value - bottom) / (top - bottom || 1);

  const coords = values.map((val, i) => {
    const x = ((i / (count - 1 || 1)) * 100).toFixed(2);
    const y = ((1 - position(val)) * 100).toFixed(2);
    return { x, y, val, pos: position(val) };
  });

  const linePath = coords
    .map((c, i) => `${i === 0 ? "M" : "L"} ${c.x} ${c.y}`)
    .join(" ");
  const areaPath = `${linePath} L 100 100 L 0 100 Z`;

  let lines = "";
  let axis = "";
  for (let tick = bottom; tick <= top; tick += step) {
    const at = position(tick);
    lines += `<span class="mc-chart-line" style="--at: ${at.toFixed(3)}"></span>`;
    axis += `<span>${tick}${unit ? " " + unit : ""}</span>`;
  }

  const days = coords
    .map((c, i) => {
      const formattedDate = formatFullDate(data[i]?.date || "");
      const tipText = `${c.val} ${unit}`.trim();
      return `<span class="about-day-col" style="left: ${c.x}%; --at: ${
        c.pos.toFixed(3)
      };" data-tip-title="${formattedDate}" data-tip="${tipText}"></span>`;
    })
    .join("");

  const tickStep = count > 14 ? 6 : count > 7 ? 3 : 2;
  const filteredTicks = data
    .map((d, i) => {
      const isLast = i === count - 1;
      const isFirst = i === 0;
      if (!isFirst && !isLast && (count - 1 - i) % tickStep !== 0) {
        return "";
      }
      return `<span>${formatShortDate(d.date)}</span>`;
    })
    .filter(Boolean)
    .join("");

  container.innerHTML = `
    <div class="mc-chart" style="--days: ${count}">
      <div class="mc-chart-axis" aria-hidden="true">${axis}</div>
      <div class="mc-chart-plot" role="img">
        ${lines}
        <svg class="mc-chart-svg" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
          <path class="mc-chart-area" d="${areaPath}" />
          <path class="mc-chart-path" d="${linePath}" />
        </svg>
        ${days}
      </div>
      <div class="about-chart-dates" aria-hidden="true">${filteredTicks}</div>
    </div>
  `;
}

function formatShortDate(dateStr: string): string {
  if (!dateStr) return "";
  const parts = dateStr.split("-");
  return parts.length === 3 ? `${parts[2]}.${parts[1]}` : dateStr;
}

function formatFullDate(dateStr: string): string {
  if (!dateStr) return "";
  const parts = dateStr.split("-");
  if (parts.length !== 3) return dateStr;
  const months = ["янв", "фев", "мар", "апр", "мая", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"];
  const mIndex = parseInt(parts[1], 10) - 1;
  const day = parseInt(parts[2], 10);
  return `${day} ${months[mIndex] || parts[1]}`;
}

function generateSampleData(days: number): DailyPoint[] {
  const list: DailyPoint[] = [];
  const now = new Date();
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(d.getDate() - i);
    const dateStr = d.toISOString().slice(0, 10);
    const baseHours = 45 + Math.sin(i / 3) * 15 + Math.floor(Math.random() * 8);
    const baseUsers = 180 + Math.cos(i / 2) * 35
      + Math.floor(Math.random() * 15);
    list.push({
      date: dateStr,
      total_seconds: Math.round(baseHours * 3600),
      active_users: Math.round(baseUsers),
    });
  }
  return list;
}
