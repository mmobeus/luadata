let ready = false;
let convertFn = null;
const pageOptions = JSON.parse(document.body.dataset.options || "{}");

// Set up interactive UI immediately
document.querySelectorAll(".code-block-lua").forEach(setupBlock);

// Load WASM in background — Run button works once this completes
loadWasm();

async function loadWasm() {
    try {
        const mod = await import("../luadata.js");
        await mod.init("pkg");
        convertFn = mod.convert;
        ready = true;
    } catch (e) {
        console.warn("WASM not available:", e.message);
    }
}

function setupBlock(block) {
    const codeEl = block.querySelector("code.language-lua");
    if (!codeEl) return;

    const original = codeEl.textContent;

    // Find the nearest following output block — skip if none found
    // or if the output isn't JSON (e.g. raw text from a Go program)
    const { output: outputBlock, schema: schemaBlock } = findRelatedBlocks(block);
    if (!outputBlock) return;
    const outputText = outputBlock.querySelector("pre").textContent.trim();
    if (!outputText.startsWith("{") && !outputText.startsWith("[")) return;
    const originalOutput = outputBlock.querySelector("pre").innerHTML;

    // Set up schema block editing if one exists
    let schemaEditor = null;
    if (schemaBlock) {
        schemaEditor = setupSchemaEditor(schemaBlock);
    }

    // Add interactive toolbar
    const toolbar = document.createElement("div");
    toolbar.className = "interactive-toolbar";

    const runBtn = document.createElement("button");
    runBtn.className = "run-btn";
    runBtn.textContent = "Run";
    runBtn.title = "Convert Lua to JSON (Ctrl+Enter)";

    const resetBtn = document.createElement("button");
    resetBtn.className = "reset-btn";
    resetBtn.textContent = "Reset";
    resetBtn.title = "Restore original example";
    resetBtn.style.display = "none";

    toolbar.appendChild(runBtn);
    toolbar.appendChild(resetBtn);

    // Create textarea (hidden initially, shown on first edit)
    const textarea = document.createElement("textarea");
    textarea.className = "lua-editor";
    textarea.value = original;
    textarea.spellcheck = false;
    textarea.style.display = "none";

    // Append textarea then toolbar so buttons are always below the editor
    block.appendChild(textarea);
    block.appendChild(toolbar);

    const preEl = block.querySelector("pre");

    // Click the code to start editing
    preEl.addEventListener("click", startEditing);
    preEl.style.cursor = "pointer";
    preEl.title = "Click to edit";

    let editing = false;

    function startEditing() {
        if (editing) return;
        editing = true;
        preEl.style.display = "none";
        textarea.style.display = "block";
        textarea.focus();
        resetBtn.style.display = "";
        autoResize();
    }

    function autoResize() {
        textarea.style.height = "auto";
        textarea.style.height = textarea.scrollHeight + "px";
    }

    textarea.addEventListener("input", autoResize);

    function buildRunOptions() {
        if (!schemaEditor) return pageOptions;
        const schema = schemaEditor.getValue();
        if (schema === null) return pageOptions;
        // Merge edited schema into page options
        const opts = Object.assign({}, pageOptions);
        opts.schema = schema;
        return opts;
    }

    // Run conversion
    runBtn.addEventListener("click", () => {
        if (!ready) {
            updateOutput(outputBlock, "Loading WASM module\u2026", true);
            return;
        }
        const input = editing ? textarea.value : original;
        try {
            const json = convertFn(input, buildRunOptions());
            const pretty = JSON.stringify(JSON.parse(json), null, 2);
            updateOutput(outputBlock, pretty, false);
        } catch (e) {
            updateOutput(outputBlock, e.message, true);
        }
    });

    // Reset to original
    resetBtn.addEventListener("click", () => {
        textarea.value = original;
        outputBlock.querySelector("pre").innerHTML = originalOutput;
        // Re-highlight if it had hljs
        const code = outputBlock.querySelector("code.hljs");
        if (code) {
            code.classList.remove("hljs");
            hljs.highlightElement(code);
        }
        textarea.style.display = "none";
        preEl.style.display = "";
        preEl.style.cursor = "pointer";
        preEl.title = "Click to edit";
        editing = false;
        resetBtn.style.display = "none";
        if (schemaEditor) schemaEditor.reset();
    });

    // Ctrl/Cmd+Enter to run
    textarea.addEventListener("keydown", (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
            e.preventDefault();
            runBtn.click();
        }
        // Tab inserts spaces
        if (e.key === "Tab") {
            e.preventDefault();
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            textarea.value =
                textarea.value.substring(0, start) +
                "    " +
                textarea.value.substring(end);
            textarea.selectionStart = textarea.selectionEnd = start + 4;
        }
    });
}

function setupSchemaEditor(block) {
    const codeEl = block.querySelector("code.language-json");
    if (!codeEl) return null;

    const original = codeEl.textContent;
    const preEl = block.querySelector("pre");

    block.classList.add("code-block-editable");

    const textarea = document.createElement("textarea");
    textarea.className = "schema-editor";
    textarea.value = original;
    textarea.spellcheck = false;
    textarea.style.display = "none";

    block.appendChild(textarea);

    preEl.addEventListener("click", startEditing);
    preEl.style.cursor = "pointer";
    preEl.title = "Click to edit schema";

    let editing = false;

    function startEditing() {
        if (editing) return;
        editing = true;
        preEl.style.display = "none";
        textarea.style.display = "block";
        textarea.focus();
        autoResize();
    }

    function autoResize() {
        textarea.style.height = "auto";
        textarea.style.height = textarea.scrollHeight + "px";
    }

    textarea.addEventListener("input", autoResize);

    // Tab inserts spaces
    textarea.addEventListener("keydown", (e) => {
        if (e.key === "Tab") {
            e.preventDefault();
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            textarea.value =
                textarea.value.substring(0, start) +
                "  " +
                textarea.value.substring(end);
            textarea.selectionStart = textarea.selectionEnd = start + 2;
        }
    });

    return {
        getValue() {
            if (!editing) return null;
            return textarea.value.trim();
        },
        reset() {
            textarea.value = original;
            textarea.style.display = "none";
            preEl.style.display = "";
            preEl.style.cursor = "pointer";
            preEl.title = "Click to edit schema";
            // Re-highlight
            const code = block.querySelector("code.hljs");
            if (code) {
                code.textContent = original;
                code.classList.remove("hljs");
                code.removeAttribute("data-highlighted");
                hljs.highlightElement(code);
            }
            editing = false;
        },
    };
}

function findRelatedBlocks(block) {
    let output = null;
    let schema = null;

    function checkForSchema(el) {
        if (!schema && el.classList.contains("code-block") &&
            !el.classList.contains("code-block-lua") &&
            el.querySelector("code.language-json")) {
            schema = el;
        }
    }

    // First check siblings within the same .col-code container
    let el = block.nextElementSibling;
    while (el) {
        if (el.classList.contains("output-block")) { output = el; break; }
        if (el.classList.contains("code-block-lua")) break;
        checkForSchema(el);
        el = el.nextElementSibling;
    }

    if (output) return { output, schema };

    // Walk forward through subsequent .row elements
    let row = block.closest(".row");
    if (!row) return { output: null, schema: null };
    let nextRow = row.nextElementSibling;
    while (nextRow) {
        if (!nextRow.classList.contains("row")) {
            nextRow = nextRow.nextElementSibling;
            continue;
        }
        // Stop if this row has another Lua block
        if (nextRow.querySelector(".code-block-lua")) break;
        const out = nextRow.querySelector(".output-block");
        if (out) { output = out; break; }
        // Check for schema block in this row
        const jsonBlock = nextRow.querySelector(".code-block:not(.code-block-lua)");
        if (jsonBlock && jsonBlock.querySelector("code.language-json")) {
            schema = jsonBlock;
        }
        nextRow = nextRow.nextElementSibling;
    }
    return { output, schema };
}

function updateOutput(outputBlock, text, isError) {
    const pre = outputBlock.querySelector("pre");
    const existingCode = pre.querySelector("code");

    if (isError) {
        pre.innerHTML = "";
        const errSpan = document.createElement("span");
        errSpan.className = "output-error";
        errSpan.textContent = text;
        pre.appendChild(errSpan);
        return;
    }

    if (existingCode) {
        existingCode.textContent = text;
        existingCode.classList.remove("hljs");
        existingCode.removeAttribute("data-highlighted");
        hljs.highlightElement(existingCode);
    } else {
        const code = document.createElement("code");
        code.className = "language-json";
        code.textContent = text;
        pre.innerHTML = "";
        pre.appendChild(code);
        hljs.highlightElement(code);
    }
}
