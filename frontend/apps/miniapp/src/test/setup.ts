import "@testing-library/jest-dom/vitest";
import React from "react";
import { afterEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
});

vi.mock("@tarojs/components", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) =>
    React.createElement("button", props),
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) =>
    React.createElement("input", props),
  Text: (props: React.HTMLAttributes<HTMLSpanElement>) =>
    React.createElement("span", props),
  View: (props: React.HTMLAttributes<HTMLDivElement>) =>
    React.createElement("div", props)
}));
