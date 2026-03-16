import { init, convert } from "./luadata.js";

init().then(() => {
    document.getElementById("status").textContent = "Ready.";
    document.getElementById("convert-btn").disabled = false;
}).catch((err) => {
    document.getElementById("status").textContent = "Failed to load WASM: " + err;
});

// Toggle visibility of dependent fields
const arrayModeEl = document.getElementById("opt-array-mode");
const maxGapEl = document.getElementById("opt-array-max-gap");
const lblMaxGap = document.getElementById("lbl-array-max-gap");

const stringMaxLenEl = document.getElementById("opt-string-max-len");
const stringModeEl = document.getElementById("opt-string-mode");
const lblStringMode = document.getElementById("lbl-string-mode");
const stringReplacementEl = document.getElementById("opt-string-replacement");
const lblStringReplacement = document.getElementById("lbl-string-replacement");

function updateArrayFields() {
    const show = arrayModeEl.value === "sparse";
    maxGapEl.classList.toggle("hidden", !show);
    lblMaxGap.classList.toggle("hidden", !show);
}

function updateStringFields() {
    const maxLen = parseInt(stringMaxLenEl.value) || 0;
    const showMode = maxLen > 0;
    stringModeEl.classList.toggle("hidden", !showMode);
    lblStringMode.classList.toggle("hidden", !showMode);

    const showReplacement = showMode && stringModeEl.value === "replace";
    stringReplacementEl.classList.toggle("hidden", !showReplacement);
    lblStringReplacement.classList.toggle("hidden", !showReplacement);
}

arrayModeEl.addEventListener("change", updateArrayFields);
stringMaxLenEl.addEventListener("input", updateStringFields);
stringModeEl.addEventListener("change", updateStringFields);

// Set initial state
updateArrayFields();
updateStringFields();

function buildOptions() {
    const opts = {};

    const emptyTable = document.getElementById("opt-empty-table").value;
    if (emptyTable !== "null") opts.emptyTable = emptyTable;

    const arrayMode = arrayModeEl.value;
    if (arrayMode !== "sparse") {
        opts.arrayMode = arrayMode;
    } else {
        const maxGap = parseInt(maxGapEl.value);
        if (maxGap !== 20) {
            opts.arrayMode = "sparse";
            opts.arrayMaxGap = maxGap;
        }
    }

    const maxLen = parseInt(stringMaxLenEl.value) || 0;
    if (maxLen > 0) {
        opts.stringTransform = {
            maxLen: maxLen,
            mode: stringModeEl.value,
        };
        if (stringModeEl.value === "replace") {
            opts.stringTransform.replacement = stringReplacementEl.value;
        }
    }

    return Object.keys(opts).length > 0 ? opts : undefined;
}

function doConvert() {
    const input = document.getElementById("lua-input").value;
    if (!input.trim()) return;

    const errorEl = document.getElementById("error");
    const outputEl = document.getElementById("json-output");
    errorEl.textContent = "";
    outputEl.value = "";

    try {
        const result = convert(input, buildOptions());
        try {
            outputEl.value = JSON.stringify(JSON.parse(result), null, 2);
        } catch {
            outputEl.value = result;
        }
    } catch (err) {
        errorEl.textContent = err.message;
    }
}

document.getElementById("convert-btn").addEventListener("click", doConvert);

// Auto-convert when options change
document.getElementById("opt-empty-table").addEventListener("change", doConvert);
arrayModeEl.addEventListener("change", doConvert);
maxGapEl.addEventListener("input", doConvert);
stringMaxLenEl.addEventListener("input", doConvert);
stringModeEl.addEventListener("change", doConvert);
stringReplacementEl.addEventListener("input", doConvert);
