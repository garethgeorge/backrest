import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { render, RenderResult } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider } from "next-themes";
import React, { useCallback, useMemo, useState } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { vi, type Mock } from "vitest";
import { Config } from "../../gen/ts/v1/config_pb";
import { Toaster } from "../components/ui/toaster";
import { ModalContextProvider } from "../components/common/ModalManager";
import { ConfigContext } from "../app/provider";

export interface RenderOptions {
  /** Value injected into ConfigContext (null = not yet loaded). */
  config?: Config | null;
  /** Initial route for the MemoryRouter, default "/". */
  route?: string;
  /** Optional route pattern to mount the UI under (e.g. "/plan/:planId"). */
  path?: string;
  /**
   * When true, setConfig updates the context (stateful wrapper) in addition to
   * being recorded on the returned spy — needed by suites that assert re-render
   * behavior after a component stores a new config (e.g. AuthenticationBoundary).
   */
  statefulConfig?: boolean;
}

export type SetConfigSpy = Mock<(c: Config) => void>;

export interface RenderWithProvidersResult extends RenderResult {
  user: ReturnType<typeof userEvent.setup>;
  setConfig: SetConfigSpy;
}

const Providers = (props: {
  children: React.ReactNode;
  options: RenderOptions;
  setConfigSpy: SetConfigSpy;
}) => {
  const { options, setConfigSpy } = props;
  const [config, setConfigState] = useState<Config | null>(
    options.config ?? null,
  );
  // Stable identities, mirroring the real provider.tsx (useState dispatch):
  // an unstable context value causes consumers keyed on setConfig identity
  // (e.g. AuthenticationBoundary's loadConfig useCallback) to re-fire effects.
  const setConfig = useCallback(
    (c: Config) => {
      setConfigSpy(c);
      if (options.statefulConfig) setConfigState(c);
    },
    [setConfigSpy, options.statefulConfig],
  );
  const configValue = options.statefulConfig
    ? config
    : (options.config ?? null);
  const ctxValue = useMemo<[Config | null, (c: Config) => void]>(
    () => [configValue, setConfig],
    [configValue, setConfig],
  );
  const content = options.path ? (
    <Routes>
      <Route path={options.path} element={props.children} />
    </Routes>
  ) : (
    props.children
  );
  return (
    <ChakraProvider value={defaultSystem}>
      <ThemeProvider attribute="class" disableTransitionOnChange>
        <ConfigContext.Provider value={ctxValue}>
          <Toaster />
          <ModalContextProvider>
            <MemoryRouter initialEntries={[options.route ?? "/"]}>
              {content}
            </MemoryRouter>
          </ModalContextProvider>
        </ConfigContext.Provider>
      </ThemeProvider>
    </ChakraProvider>
  );
};

/**
 * Renders under the full provider stack (Chakra + theme + ConfigContext +
 * ModalManager + MemoryRouter + Toaster). Deliberately does NOT use
 * AppProvider: its window-focus listener hits backrestService.getConfig and
 * its config state is not injectable.
 */
export const renderWithProviders = (
  ui: React.ReactNode,
  options: RenderOptions = {},
): RenderWithProvidersResult => {
  const setConfigSpy: SetConfigSpy = vi.fn();
  const result = render(
    <Providers options={options} setConfigSpy={setConfigSpy}>
      {ui}
    </Providers>,
  );
  return {
    ...result,
    user: userEvent.setup(),
    setConfig: setConfigSpy,
  };
};

/**
 * Variant for suites that combine fake timers with user interaction: user-event
 * must advance the fake clock or its internal delays deadlock. Call
 * vi.useFakeTimers() BEFORE this.
 */
export const renderWithFakeTimerUser = (
  ui: React.ReactNode,
  options: RenderOptions = {},
): RenderWithProvidersResult => {
  const setConfigSpy: SetConfigSpy = vi.fn();
  const result = render(
    <Providers options={options} setConfigSpy={setConfigSpy}>
      {ui}
    </Providers>,
  );
  return {
    ...result,
    user: userEvent.setup({ advanceTimers: vi.advanceTimersByTimeAsync }),
    setConfig: setConfigSpy,
  };
};
