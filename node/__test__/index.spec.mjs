import { convertLuaToJson, convertLuaFileToJson } from "../index.js";
import { strict as assert } from "node:assert";

// Basic conversion
const json = convertLuaToJson('playerName = "Thrall"');
const data = JSON.parse(json);
assert.equal(data.playerName, "Thrall");

// With options
const json2 = convertLuaToJson('items = {}', { emptyTable: "array" });
const data2 = JSON.parse(json2);
assert.deepEqual(data2.items, []);

// File conversion
import { writeFileSync, unlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const tmpFile = join(tmpdir(), "luadata-napi-test.lua");
writeFileSync(tmpFile, 'level = 60');
try {
    const json3 = convertLuaFileToJson(tmpFile);
    const data3 = JSON.parse(json3);
    assert.equal(data3.level, 60);
} finally {
    unlinkSync(tmpFile);
}

// Error handling
assert.throws(() => convertLuaToJson("invalid = {"), /parse/i);

console.log("All napi tests passed.");
