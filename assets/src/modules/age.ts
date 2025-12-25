declare global {
  interface Window {
    isAdultContent?: boolean;
  }
}

export function initAgeGate(): void {
  if (!window.isAdultContent) return;

  const modal = document.getElementById("age-modal");
  const confirmBtn = document.getElementById("age-confirm");
  const backBtn = document.getElementById("age-back");
  const rememberChk = document.getElementById(
    "age-remember",
  ) as HTMLInputElement | null;

  if (!modal || !confirmBtn || !backBtn || !rememberChk) {
    console.warn("Age gate elements missing in DOM");
    return;
  }

  const isConfirmed =
    localStorage.getItem("ageConfirmed") === "true" ||
    sessionStorage.getItem("ageConfirmed") === "true";

  if (!isConfirmed) {
    console.info("Age verification required, showing modal");
    modal.style.display = "flex";
    document.body.style.overflow = "hidden";
  } else {
    console.info("Age verification already confirmed");
  }

  confirmBtn.onclick = () => {
    console.info(`Age confirmed. Remember: ${rememberChk.checked}`);
    if (rememberChk.checked) {
      localStorage.setItem("ageConfirmed", "true");
    } else {
      sessionStorage.setItem("ageConfirmed", "true");
    }
    modal.style.display = "none";
    document.body.style.overflow = "";
  };

  backBtn.onclick = () => {
    console.warn("Age verification declined, redirecting home");
    window.location.href = "/";
  };
}
