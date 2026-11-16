import assert from "node:assert/strict";
import test from "node:test";
import { applyFormat, richText, toggleBullets } from "./email-design.ts";

test("one text block keeps paragraphs and bullets together", () => {
  const html = richText("Hello\n\n- First\n- Second\n\nThanks");
  assert.match(html, /Hello/);
  assert.match(html, /<ul/);
  assert.match(html, /<li[^>]*>First<\/li>/);
  assert.match(html, /<li[^>]*>Second<\/li>/);
  assert.match(html, /Thanks/);
  assert.equal(html.includes("<ul"), true);
});

test("bold wraps the selection and a second click removes it", () => {
  const wrapped = applyFormat("Hello", 0, 5, "**");
  assert.equal(wrapped.value, "**Hello**");
  const plain = applyFormat(wrapped.value, 0, wrapped.value.length, "**");
  assert.equal(plain.value, "Hello");
});

test("bullet button marks the current lines", () => {
  const marked = toggleBullets("One\nTwo", 0, 7);
  assert.equal(marked.value, "- One\n- Two");
  const html = richText(marked.value);
  assert.match(html, /<ul/);
  assert.doesNotMatch(html, /<li[^>]*>- /);
});
