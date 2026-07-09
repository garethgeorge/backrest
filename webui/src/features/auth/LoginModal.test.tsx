import { screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as m from "../../paraglide/messages";
import { LoginModal } from "./LoginModal";
import {
  renderWithProviders,
  renderWithFakeTimerUser,
} from "../../test/render";
import { authenticationService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import { connectError, Code } from "../../test/proto";
import { LoginResponseSchema } from "../../../gen/ts/v1/authentication_pb";

/**
 * user-event's default (non-null) delay makes it schedule its own internal
 * waits via fake timers; under this vitest/user-event combo those waits never
 * self-advance even though `advanceTimers: vi.advanceTimersByTimeAsync` is
 * wired up in renderWithFakeTimerUser. Pump the fake clock in small steps
 * alongside the interaction so it can settle, while staying well under the
 * component's 500ms reload delay so we never trigger window.location.reload().
 */
const flushUserEvent = async <T,>(promise: Promise<T>): Promise<T> => {
  let settled = false;
  let result: T | undefined;
  let failure: unknown;
  promise.then(
    (r) => {
      result = r;
      settled = true;
    },
    (e) => {
      failure = e;
      settled = true;
    },
  );
  for (let i = 0; i < 20 && !settled; i++) {
    await vi.advanceTimersByTimeAsync(10);
  }
  if (!settled) {
    throw new Error("user-event action did not settle under fake timers");
  }
  if (failure) throw failure;
  return result as T;
};

describe("LoginModal", () => {
  it("renders username/password inputs and the login button", async () => {
    renderWithProviders(<LoginModal />);

    expect(
      await screen.findByPlaceholderText(m.login_username_placeholder()),
    ).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(m.login_password_placeholder()),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: m.login_button() }),
    ).toBeInTheDocument();
  });

  it("shows a username-required error and does not call login when both fields are empty", async () => {
    const errorSpy = vi.spyOn(alerts, "error");
    const { user } = renderWithProviders(<LoginModal />);

    const loginButton = await screen.findByRole("button", {
      name: m.login_button(),
    });
    await user.click(loginButton);

    expect(errorSpy).toHaveBeenCalled();
    const [content] = errorSpy.mock.calls[0];
    expect(String(content)).toContain(m.login_username_required());
    expect(authenticationService.login).not.toHaveBeenCalled();
  });

  it("shows a password-required error when only username is filled", async () => {
    const errorSpy = vi.spyOn(alerts, "error");
    const { user } = renderWithProviders(<LoginModal />);

    const usernameInput = await screen.findByPlaceholderText(
      m.login_username_placeholder(),
    );
    await user.type(usernameInput, "someuser");

    const loginButton = screen.getByRole("button", { name: m.login_button() });
    await user.click(loginButton);

    expect(errorSpy).toHaveBeenCalled();
    const [content] = errorSpy.mock.calls[0];
    expect(String(content)).toContain(m.login_password_required());
    expect(authenticationService.login).not.toHaveBeenCalled();
  });

  describe("with fake timers", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("logs in successfully, stores the token, and shows a success toast", async () => {
      vi.useFakeTimers();
      const successSpy = vi.spyOn(alerts, "success");
      vi.mocked(authenticationService.login).mockResolvedValue(
        create(LoginResponseSchema, { token: "tok-123" }),
      );

      const { user } = renderWithFakeTimerUser(<LoginModal />);

      const usernameInput = screen.getByPlaceholderText(
        m.login_username_placeholder(),
      );
      const passwordInput = screen.getByPlaceholderText(
        m.login_password_placeholder(),
      );
      await flushUserEvent(user.type(usernameInput, "someuser"));
      await flushUserEvent(user.type(passwordInput, "secret-password"));

      const loginButton = screen.getByRole("button", {
        name: m.login_button(),
      });
      await flushUserEvent(user.click(loginButton));

      expect(authenticationService.login).toHaveBeenCalledWith(
        expect.objectContaining({
          username: "someuser",
          password: "secret-password",
        }),
      );
      expect(localStorage.getItem("backrest-ui-authToken")).toBe("tok-123");
      expect(successSpy).toHaveBeenCalledWith(m.login_success());
    });

    it("shows an invalid-credentials error on rejection and re-enables the login button", async () => {
      vi.useFakeTimers();
      const errorSpy = vi.spyOn(alerts, "error");
      vi.mocked(authenticationService.login).mockRejectedValue(
        connectError(Code.Unauthenticated),
      );

      const { user } = renderWithFakeTimerUser(<LoginModal />);

      const usernameInput = screen.getByPlaceholderText(
        m.login_username_placeholder(),
      );
      const passwordInput = screen.getByPlaceholderText(
        m.login_password_placeholder(),
      );
      await flushUserEvent(user.type(usernameInput, "someuser"));
      await flushUserEvent(user.type(passwordInput, "wrong-password"));

      const loginButton = screen.getByRole("button", {
        name: m.login_button(),
      });
      await flushUserEvent(user.click(loginButton));

      expect(errorSpy).toHaveBeenCalled();
      const [content] = errorSpy.mock.calls[errorSpy.mock.calls.length - 1];
      expect(String(content)).toContain(m.login_password_invalid());
      expect(loginButton).not.toBeDisabled();
    });

    it("submits the form when Enter is pressed in the password field", async () => {
      vi.useFakeTimers();
      vi.mocked(authenticationService.login).mockResolvedValue(
        create(LoginResponseSchema, { token: "tok-456" }),
      );

      const { user } = renderWithFakeTimerUser(<LoginModal />);

      const usernameInput = screen.getByPlaceholderText(
        m.login_username_placeholder(),
      );
      const passwordInput = screen.getByPlaceholderText(
        m.login_password_placeholder(),
      );
      await flushUserEvent(user.type(usernameInput, "someuser"));
      await flushUserEvent(user.type(passwordInput, "secret-password{Enter}"));

      expect(authenticationService.login).toHaveBeenCalledTimes(1);
    });
  });
});
