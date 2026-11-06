"use strict";

function validVerificationURL(value) {
  const parsed = new URL(value);
  const local = ["localhost", "127.0.0.1", "[::1]"].includes(parsed.hostname);
  if (!(parsed.protocol === "https:" || (local && parsed.protocol === "http:")) ||
      parsed.username || parsed.password || parsed.search ||
      parsed.pathname !== "/verify.html" ||
      !/^#identity=[A-Za-z0-9_-]{32,128}$/.test(parsed.hash)) {
    throw new Error("URL de verificação temporária inválida.");
  }
  return parsed.href;
}

function report(message) {
  document.getElementById("error").textContent = message;
}

// Liveness has no subject reference; do not require an enrollment identifier.
const flowSelector = document.getElementById("flow");
const subjectInput = document.getElementById("subject-id");
function updateSubjectInput() {
  const needsSubject = flowSelector.value !== "liveness";
  subjectInput.required = needsSubject;
  subjectInput.disabled = !needsSubject;
}
flowSelector.addEventListener("change", updateSubjectInput);
updateSubjectInput();

const statusButton = document.getElementById("status-button");
const statusText = document.getElementById("check-status");
let lastCheck = null;

function rememberCheck(id, flow) {
  if (!/^chk_[A-Za-z0-9_-]{16}$/.test(id) || !["liveness","enroll","verify"].includes(flow)) {
    throw new Error("Identificador de verificação inesperado.");
  }
  lastCheck = {checkId:id,flow};
  statusButton.disabled = false;
  statusText.textContent = "Captura iniciada. Após concluí-la, consulte o resultado nesta página.";
  try { sessionStorage.setItem("ifacematch:local-demo-check", JSON.stringify(lastCheck)); }
  catch (_) { /* The current page still supports the status action. */ }
}

try {
  const saved = JSON.parse(sessionStorage.getItem("ifacematch:local-demo-check") || "null");
  if (saved && /^chk_[A-Za-z0-9_-]{16}$/.test(saved.checkId) &&
      ["liveness","enroll","verify"].includes(saved.flow)) {
    lastCheck = saved;
    statusButton.disabled = false;
    statusText.textContent = "Último check " + saved.checkId + ": consulte a decisão do Server.";
  }
} catch (_) { /* No local browser storage is required. */ }

statusButton.addEventListener("click", async () => {
  if (!lastCheck) return;
  statusButton.disabled = true;
  report("");
  try {
    const response = await fetch("/status", {
      method:"POST",credentials:"same-origin",cache:"no-store",
      headers:{"Content-Type":"application/json"},
      body:JSON.stringify(lastCheck)
    });
    if (!response.ok) throw new Error("Consulta indisponível.");
    const result = await response.json();
    if (result.id !== lastCheck.checkId || typeof result.status !== "string")
      throw new Error("Resposta inesperada.");
    const decision = typeof result.decision === "string" && result.decision
      ? " — decisão: " + result.decision : "";
    statusText.textContent = "Check " + result.id + ": " + result.status + decision;
  } catch (_) {
    report("Não foi possível consultar o resultado no Server. Confira se o backend local e a aplicação continuam disponíveis.");
  } finally {
    statusButton.disabled = false;
  }
});

document.getElementById("issue-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  report("");
  const button = event.currentTarget.querySelector("button");
  button.disabled = true;
  try {
    const response = await fetch("/start", {
      method: "POST",
      credentials: "same-origin",
      cache: "no-store",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({
        flow: document.getElementById("flow").value,
        subjectId: document.getElementById("subject-id").value.trim()
      })
    });
    if (!response.ok) throw new Error("Falha ao emitir a verificação pelo backend.");
    const result = await response.json();
    const verificationUrl = validVerificationURL(result.verificationUrl);
    rememberCheck(result.id, document.getElementById("flow").value);
    window.location.assign(verificationUrl);
  } catch (_) {
    report("Para emitir uma verificação, execute o exemplo Go localmente com uma chave de aplicação válida. Consulte a documentação.");
  } finally {
    button.disabled = false;
  }
});

document.getElementById("verification-form").addEventListener("submit", (event) => {
  event.preventDefault();
  try {
    window.location.assign(validVerificationURL(document.getElementById("verification-url").value.trim()));
  } catch (_) {
    report("Cole uma URL temporária válida emitida pelo iFaceMatch.");
  }
});
