"use strict";

const copyButtons = document.querySelectorAll("[data-copy-code]");

for (const button of copyButtons) {
  button.addEventListener("click", async () => {
    const code = button.closest(".code-shell")?.querySelector("pre code");

    if (!code) {
      return;
    }

    try {
      await navigator.clipboard.writeText(code.textContent ?? "");
      button.textContent = "Copiado";
    } catch {
      button.textContent = "Selecione o código para copiar";
    }

    window.setTimeout(() => {
      button.textContent = "Copiar";
    }, 1800);
  });
}

const sidebar = document.querySelector("[data-sidebar]");
const backdrop = document.querySelector("[data-mobile-backdrop]");
const menuButton = document.querySelector("[data-menu-toggle]");

function setMenuOpen(isOpen) {
  sidebar?.classList.toggle("open", isOpen);
  backdrop?.classList.toggle("open", isOpen);
  menuButton?.setAttribute("aria-expanded", String(isOpen));
}

menuButton?.addEventListener("click", () => {
  setMenuOpen(!sidebar?.classList.contains("open"));
});

backdrop?.addEventListener("click", () => {
  setMenuOpen(false);
});

sidebar?.querySelectorAll("a").forEach((link) => {
  link.addEventListener("click", () => setMenuOpen(false));
});

document.addEventListener("htmx:afterSwap", (event) => {
  const target = event.detail?.target;

  if (target?.id === "search-results") {
    target.classList.toggle("open", Boolean(target.textContent?.trim()));
  }
});

document.addEventListener("click", (event) => {
  const search = document.querySelector(".search-wrap");

  if (search && !search.contains(event.target)) {
    document.querySelector("#search-results")?.classList.remove("open");
  }
});

document.addEventListener("keydown", (event) => {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    document.querySelector("#documentation-search")?.focus();
  }

  if (event.key === "Escape") {
    setMenuOpen(false);
    document.querySelector("#search-results")?.classList.remove("open");
  }
});
