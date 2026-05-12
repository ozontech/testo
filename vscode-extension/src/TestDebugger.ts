import * as vscode from "vscode";

export interface DebugSuiteTestParams {
    readonly workspace?: vscode.WorkspaceFolder;
    readonly dir: string;
    readonly suiteCallerTest: string;
    readonly suiteName: string;
    readonly testName: string;
    readonly tags: string;
    readonly envs: Map<string, string>;
    readonly extraArgs: string[];
    readonly extraBuildArgs: string[];
}

export class TestDebugger {
    public static async debugSuiteTest(params: DebugSuiteTestParams) {
        const buildFlags = params.tags ? ["-tags", params.tags] : [];

        buildFlags.push(...params.extraBuildArgs);

        const args = [
            "-test.run",
            `^${params.suiteCallerTest}$/^${params.suiteName}$`,
            "-testo.m",
            `^${params.testName}$`,
        ];

        args.push(...params.extraArgs);

        const debugConfig: vscode.DebugConfiguration = {
            name: "Debug Testo Test",
            type: "go",
            request: "launch",
            mode: "test",
            program: params.dir,
            env: params.envs,
            args,
            buildFlags: buildFlags.join(" "),
        };

        if (
            vscode.workspace
                .getConfiguration()
                .get("debug.internalConsoleOptions") !== "neverOpen"
        ) {
            vscode.commands.executeCommand("workbench.debug.action.focusRepl");
        }

        return await vscode.debug.startDebugging(params.workspace, debugConfig);
    }
}
