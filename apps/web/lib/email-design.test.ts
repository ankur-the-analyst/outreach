import assert from "node:assert/strict";
import test from "node:test";
import {
  applyFormat,
  applyKind,
  formatKindFromKey,
  historyPush,
  historyRedo,
  historyStart,
  historyUndo,
  richText,
  toggleBullets,
  wrapLink,
} from "./email-design.ts";

test("one text block keeps paragraphs and bullets together", () => {
  const html = richText("Hello\n\n- First\n- Second\n\nThanks");
  assert.match(html, /Hello/);
  assert.match(html, /<ul/);
  assert.match(html, /<li[^>]*>First<\/li>/);
  assert.match(html, /<li[^>]*>Second<\/li>/);
  assert.match(html, /Thanks/);
});

test("a wrapped bullet stays in the same point", () => {
  const html = richText("- A long point\nthat continues\n- Next");
  assert.match(html, /<li[^>]*>A long point<br>that continues<\/li>/);
  assert.match(html, /<li[^>]*>Next<\/li>/);
  assert.equal(html.match(/<ul/g)?.length, 1);
});

test("bold wraps the selection and a second click removes it", () => {
  const wrapped = applyFormat("Hello", 0, 5, "**");
  assert.equal(wrapped.value, "**Hello**");
  const plain = applyFormat(wrapped.value, 0, wrapped.value.length, "**");
  assert.equal(plain.value, "Hello");
});

test("formatter keys map to bold, italic, and bullets", () => {
  assert.equal(formatKindFromKey("b", false, true), "bold");
  assert.equal(formatKindFromKey("i", false, true), "italic");
  assert.equal(formatKindFromKey("u", false, true), "underline");
  assert.equal(formatKindFromKey("h", true, true), "highlight");
  assert.equal(formatKindFromKey("x", true, true), "strike");
  assert.equal(formatKindFromKey("*", true, true), "bullets");
  assert.equal(formatKindFromKey("*", true, false), null);
  const wrapped = applyKind("bold", "Hello", 0, 5);
  assert.equal(wrapped.value, "**Hello**");
});

test("a link marker becomes a hyperlink", () => {
  const html = richText("See [the role](https://jobs.example/1).");
  assert.match(html, /<a href="https:\/\/jobs\.example\/1"/);
  assert.match(html, />the role<\/a>/);
});

test("undo returns the previous text and redo puts it back", () => {
  let history = historyStart("Hello");
  history = historyPush(history, "Hello there");
  history = historyUndo(history);
  assert.equal(history.stack[history.index], "Hello");
  history = historyRedo(history);
  assert.equal(history.stack[history.index], "Hello there");
  const linked = wrapLink("See more", 4, 8, "https://x.test");
  assert.equal(linked.value, "See [more](https://x.test)");
});

test("bullet button marks the current lines", () => {
  const marked = toggleBullets("One\nTwo", 0, 7);
  assert.equal(marked.value, "- One\n- Two");
  const html = richText(marked.value);
  assert.match(html, /<ul/);
  assert.doesNotMatch(html, /<li[^>]*>- /);
});
