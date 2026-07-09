import { Button } from "@chakra-ui/react";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import * as m from "../paraglide/messages";
import { LoginModal } from "../features/auth/LoginModal";
import { renderWithProviders } from "./render";

// Foundation smoke test: proves the Chakra v3 + Zag + jsdom + paraglide +
// mocked-client stack works end to end. If this file fails, fix the setup in
// src/test/ before writing component suites.
describe("test foundation", () => {
  it("renders a Chakra component with a paraglide message", () => {
    renderWithProviders(<Button>{m.login_button()}</Button>);
    expect(
      screen.getByRole("button", { name: m.login_button() }),
    ).toBeVisible();
  });

  it("renders LoginModal's dialog into a portal", async () => {
    renderWithProviders(<LoginModal />);
    // FormModal renders via a Chakra Dialog portal on document.body.
    expect(await screen.findByText(m.login_title())).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(m.login_username_placeholder()),
    ).toBeInTheDocument();
  });

  it("does not hang when importing the streaming singleton modules", async () => {
    // oplog.ts / peerStates.ts start for-await loops at import time; the global
    // client mock parks them on a never-yielding stream.
    await import("../api/oplog");
    await import("../state/peerStates");
  });
});
