import { init, convert } from "./luadata.js";

init().then(() => {
    document.getElementById("convert-btn").disabled = false;
}).catch((err) => {
    document.getElementById("error").textContent = "Failed to load WASM: " + err;
});

// Toggle visibility of dependent fields
const schemaEl = document.getElementById("opt-schema");
const unknownFieldsEl = document.getElementById("opt-unknown-fields");
const lblUnknownFields = document.getElementById("lbl-unknown-fields");

const arrayModeEl = document.getElementById("opt-array-mode");
const maxGapEl = document.getElementById("opt-array-max-gap");
const lblMaxGap = document.getElementById("lbl-array-max-gap");

const stringMaxLenEl = document.getElementById("opt-string-max-len");
const stringModeEl = document.getElementById("opt-string-mode");
const lblStringMode = document.getElementById("lbl-string-mode");
const stringReplacementEl = document.getElementById("opt-string-replacement");
const lblStringReplacement = document.getElementById("lbl-string-replacement");

function updateSchemaFields() {
    const hasSchema = schemaEl.value.trim().length > 0;
    unknownFieldsEl.classList.toggle("hidden", !hasSchema);
    lblUnknownFields.classList.toggle("hidden", !hasSchema);
}

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

schemaEl.addEventListener("input", updateSchemaFields);
arrayModeEl.addEventListener("change", updateArrayFields);
stringMaxLenEl.addEventListener("input", updateStringFields);
stringModeEl.addEventListener("change", updateStringFields);

// Set initial state
updateSchemaFields();
updateArrayFields();
updateStringFields();

function buildOptions() {
    const opts = {};

    const schema = schemaEl.value.trim();
    if (schema) {
        opts.schema = schema;
        const unknownFields = unknownFieldsEl.value;
        if (unknownFields !== "ignore") opts.unknownFields = unknownFields;
    }

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
schemaEl.addEventListener("input", doConvert);
unknownFieldsEl.addEventListener("change", doConvert);
document.getElementById("opt-empty-table").addEventListener("change", doConvert);
arrayModeEl.addEventListener("change", doConvert);
maxGapEl.addEventListener("input", doConvert);
stringMaxLenEl.addEventListener("input", doConvert);
stringModeEl.addEventListener("change", doConvert);
stringReplacementEl.addEventListener("input", doConvert);

// Example generator
const examples = [
    `playerName = "Thrall"
playerLevel = 60
isOnline = true
guild = nil`,

    `MyAddonDB = {
    ["profiles"] = {
        ["Default"] = {
            ["minimap"] = {
                ["hide"] = false,
            },
            ["fontSize"] = 12,
            ["theme"] = "dark",
        },
    },
}`,

    `QuestLog = {
    ["activeQuests"] = {
        {
            ["title"] = "The Frozen Throne",
            ["level"] = 80,
            ["objectives"] = {"Defeat the Lich King", "Survive"},
            ["rewards"] = {
                ["gold"] = 50,
                ["items"] = {"Frostmourne Shard", "Helm of Domination"},
            },
        },
        {
            ["title"] = "A New Dawn",
            ["level"] = 20,
            ["objectives"] = {"Speak with the innkeeper"},
            ["rewards"] = {},
        },
    },
    ["completedCount"] = 142,
}`,

    `GuildRoster = {
    [1] = {
        ["name"] = "Arthas",
        ["class"] = "Death Knight",
        ["level"] = 80,
        ["online"] = true,
    },
    [3] = {
        ["name"] = "Jaina",
        ["class"] = "Mage",
        ["level"] = 80,
        ["online"] = false,
    },
    [7] = {
        ["name"] = "Anduin",
        ["class"] = "Priest",
        ["level"] = 70,
        ["online"] = true,
    },
}`,

    `AuctionData = {
    ["lastScan"] = 1710547200,
    ["items"] = {
        ["Copper Ore"] = {
            ["minPrice"] = 150,
            ["avgPrice"] = 210,
            ["quantity"] = 4832,
        },
        ["Linen Cloth"] = {
            ["minPrice"] = 50,
            ["avgPrice"] = 85,
            ["quantity"] = 12041,
        },
    },
    ["scanDuration"] = 3.45,
}`,

    `BagContents = {
    [0] = {
        [1] = {["id"] = 6948, ["name"] = "Hearthstone", ["count"] = 1},
        [2] = {["id"] = 159, ["name"] = "Refreshing Spring Water", ["count"] = 20},
        [5] = {["id"] = 2589, ["name"] = "Linen Cloth", ["count"] = 12},
    },
    [1] = {
        [1] = {["id"] = 774, ["name"] = "Malachite", ["count"] = 3},
    },
}`,

    `-- Talent spec saved from inspect
TalentSpec = {
    ["name"] = "Holy",
    ["class"] = "Paladin",
    ["points"] = {
        ["Holy"] = 51,
        ["Protection"] = 5,
        ["Retribution"] = 15,
    },
    ["glyphs"] = {"Glyph of Holy Light", "Glyph of Seal of Wisdom"},
}`,

    `ChatLog = {
    ["messages"] = {
        {["channel"] = "Guild", ["sender"] = "Thrall", ["text"] = "LFM Onyxia, need heals", ["timestamp"] = 1710547200},
        {["channel"] = "Guild", ["sender"] = "Jaina", ["text"] = "omw", ["timestamp"] = 1710547215},
        {["channel"] = "Whisper", ["sender"] = "Varian", ["text"] = "inv pls", ["timestamp"] = 1710547220},
    },
    ["filters"] = {
        ["showGuild"] = true,
        ["showWhisper"] = true,
        ["showSay"] = false,
    },
}`,

    `EquipmentSet = {
    ["name"] = "Raid Healing",
    ["icon"] = "INV_Helmet_51",
    ["slots"] = {
        [1] = {["id"] = 40298, ["name"] = "Helm of the Lost Conqueror"},
        [3] = {["id"] = 40301, ["name"] = "Valorous Redemption Shoulderguards"},
        [5] = {["id"] = 40569, ["name"] = "Valorous Redemption Tunic"},
        [6] = {["id"] = 40259, ["name"] = "Waistguard of Divine Grace"},
        [7] = {["id"] = 40572, ["name"] = "Valorous Redemption Greaves"},
        [8] = {["id"] = 40592, ["name"] = "Blues"},
        [10] = {["id"] = 40399, ["name"] = "Boundless Ambition"},
    },
}`,

    `MapPins = {
    ["Elwynn Forest"] = {
        {["x"] = 48.2, ["y"] = 42.1, ["label"] = "Goldshire Inn", ["icon"] = "quest"},
        {["x"] = 32.7, ["y"] = 50.8, ["label"] = "Crystal Lake", ["icon"] = "fishing"},
    },
    ["Durotar"] = {
        {["x"] = 52.1, ["y"] = 41.3, ["label"] = "Sen'jin Village", ["icon"] = "flightpath"},
    },
    ["settings"] = {
        ["showOnMinimap"] = true,
        ["opacity"] = 0.85,
    },
}`,

    `-- DKP standings after raid
DKPStandings = {
    {["name"] = "Legolas", ["class"] = "Hunter", ["dkp"] = 245.5, ["lifetime"] = 1820},
    {["name"] = "Gandalf", ["class"] = "Mage", ["dkp"] = 312.0, ["lifetime"] = 2105},
    {["name"] = "Aragorn", ["class"] = "Warrior", ["dkp"] = 198.5, ["lifetime"] = 1540},
    {["name"] = "Gimli", ["class"] = "Warrior", ["dkp"] = 156.0, ["lifetime"] = 980},
    {["name"] = "Samwise", ["class"] = "Paladin", ["dkp"] = 421.0, ["lifetime"] = 2890},
}`,

    `WeakAuras = {
    ["displays"] = {
        ["Health Warning"] = {
            ["trigger"] = {
                ["type"] = "status",
                ["event"] = "Health",
                ["unit"] = "player",
                ["percenthealth"] = 30,
                ["use_percenthealth"] = true,
            },
            ["load"] = {
                ["combat"] = true,
            },
            ["regionType"] = "icon",
            ["color"] = {1, 0, 0, 1},
        },
    },
}`,

    `ServerConfig = {
    ["realm"] = "Proudmoore",
    ["region"] = "US",
    ["characters"] = {
        {["name"] = "Moonfire", ["level"] = 80, ["class"] = "Druid", ["race"] = "Night Elf"},
        {["name"] = "Shadowstep", ["level"] = 73, ["class"] = "Rogue", ["race"] = "Human"},
    },
    ["settings"] = {
        ["autoRepair"] = true,
        ["autoSell"] = true,
        ["sellList"] = {"Poor", "Common"},
        ["ignoreList"] = {},
    },
}`,

    `RaidEncounters = {
    ["Naxxramas"] = {
        ["Patchwerk"] = {["kills"] = 12, ["wipes"] = 3, ["bestTime"] = 185.4},
        ["Grobbulus"] = {["kills"] = 11, ["wipes"] = 7, ["bestTime"] = 210.1},
        ["Thaddius"] = {["kills"] = 9, ["wipes"] = 14, ["bestTime"] = 312.8},
    },
    ["Ulduar"] = {
        ["Flame Leviathan"] = {["kills"] = 5, ["wipes"] = 0, ["bestTime"] = 95.2},
        ["XT-002"] = {["kills"] = 4, ["wipes"] = 2, ["bestTime"] = 275.6},
    },
    ["lastUpdated"] = 1710633600,
}`,
];

let exampleIndex = 0;

document.getElementById("example-btn").addEventListener("click", () => {
    const inputEl = document.getElementById("lua-input");
    inputEl.value = examples[exampleIndex % examples.length];
    exampleIndex++;
    doConvert();
});
