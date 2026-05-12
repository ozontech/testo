import * as vscode from "vscode";

export interface RunSuiteParams {
  readonly dir: string;
  readonly verbose: boolean;
  readonly suiteCallerTest: string;
  readonly suiteName: string;
  readonly tags: string;
  readonly envs: Map<string, string>;
}

export interface RunSuiteTestParams {
  readonly dir: string;
  readonly verbose: boolean;
  readonly suiteCallerTest: string;
  readonly suiteName: string;
  readonly testName: string;
  readonly tags: string;
  readonly envs: Map<string, string>;
}

export class TestRunner {
  private static readonly terminalName = "Testo";

  public static runSuite(params: RunSuiteParams) {
    const cmd: string[] = ["env", "-C", `'${params.dir}'`];

    for (const [key, value] of params.envs) {
      cmd.push(`'${key}'='${value}'`);
    }

    cmd.push(
      "go",
      "test",
      "-run",
      `'^${params.suiteCallerTest}$/^${params.suiteName}$'`,
    );

    if (params.verbose) {
      cmd.push("-v");
    }

    if (params.tags) {
      cmd.push(`-tags='${params.tags}'`);
    }

    TestRunner.runInTerminal(cmd);
  }

  public static runSuiteTest(params: RunSuiteTestParams) {
    const cmd: string[] = ["env", "-C", `'${params.dir}'`];

    for (const [key, value] of params.envs) {
      cmd.push(`'${key}'='${value}'`);
    }

    cmd.push(
      "go",
      "test",
      "-run",
      `'^${params.suiteCallerTest}$/^${params.suiteName}$'`,
      "-testo.m",
      `'^${params.testName}$'`,
    );

    if (params.verbose) {
      cmd.push("-v");
    }

    if (params.tags) {
      cmd.push(`-tags='${params.tags}'`);
    }

    TestRunner.runInTerminal(cmd);
  }

  private static runInTerminal(args: string[]) {
    const terminal = TestRunner.terminal();

    terminal.sendText(args.join(" "));
    terminal.show(true);
  }

  private static terminal(): vscode.Terminal {
    for (const terminal of vscode.window.terminals) {
      if (terminal.name === TestRunner.terminalName) {
        return terminal;
      }
    }

    return vscode.window.createTerminal({
      name: TestRunner.terminalName,
      hideFromUser: true,
    });
  }
}
