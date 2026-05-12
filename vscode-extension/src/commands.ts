import * as vscode from "vscode";

import { DocumentScoped, Suite, SuiteTest } from "./models";
import { findSuiteCaller, goConfig, resolvePath, testoConfig } from "./utils";
import { CONFIG_VERBOSE_OUTPUT } from "./const";
import { TestRunner } from "./TestRunner";
import { TestDebugger } from "./TestDebugger";

export class Commands {
    public static async suiteRun(suite: Suite) {
        const caller = await findSuiteCaller(
            suite.doc.uri,
            suite.symbol.selectionRange.start,
        );

        if (!caller) {
            vscode.window.showErrorMessage("Suite caller not found.");

            return;
        }

        TestRunner.runSuite({
            dir: vscode.Uri.joinPath(caller.document.uri, "..").fsPath,
            verbose: testoConfig().get(CONFIG_VERBOSE_OUTPUT) ?? false,
            suiteCallerTest: caller.value.name,
            suiteName: suite.symbol.name,
            tags: getTestTags(),
            envs: getTestEnvs(),
        });
    }

    public static async suiteRunTest(
        test: SuiteTest<DocumentScoped<vscode.DocumentSymbol>>,
    ) {
        const caller = await findSuiteCaller(
            test.suite.document.uri,
            test.suite.value.selectionRange.start,
        );

        if (!caller) {
            vscode.window.showErrorMessage(
                "Suite caller of this test not found.",
            );

            return;
        }

        TestRunner.runSuiteTest({
            dir: vscode.Uri.joinPath(caller.document.uri, "..").fsPath,
            verbose: testoConfig().get(CONFIG_VERBOSE_OUTPUT) ?? false,
            suiteCallerTest: caller.value.name,
            suiteName: test.suite.value.name,
            testName: test.name,
            tags: getTestTags(),
            envs: getTestEnvs(),
        });
    }

    public static async suiteDebugTest(
        test: SuiteTest<DocumentScoped<vscode.DocumentSymbol>>,
    ) {
        const caller = await findSuiteCaller(
            test.suite.document.uri,
            test.suite.value.selectionRange.start,
        );

        if (!caller) {
            vscode.window.showErrorMessage(
                "Suite caller of this test not found.",
            );

            return;
        }

        const workspaceFolder = vscode.workspace.getWorkspaceFolder(
            test.doc.uri,
        );

        const extraArgs: string[] = [];
        const extraBuildArgs: string[] = [];

        const flagsFromConfig = getTestFlags();
        let foundArgsFlag = false;
        flagsFromConfig.forEach((x) => {
            if (foundArgsFlag) {
                extraArgs.push(x);
                return;
            }

            if (x === "-args") {
                foundArgsFlag = true;
                return;
            }

            extraBuildArgs.push(x);
        });

        TestDebugger.debugSuiteTest({
            workspace: workspaceFolder,
            dir: vscode.Uri.joinPath(caller.document.uri, "..").fsPath,
            suiteCallerTest: caller.value.name,
            suiteName: test.suite.value.name,
            testName: test.name,
            tags: getTestTags(),
            envs: getTestEnvs(),
            extraArgs,
            extraBuildArgs,
        });
    }
}

function getTestTags(): string {
    const config = goConfig();

    return config["testTags"] !== null
        ? config["testTags"]
        : config["buildTags"];
}

function getTestEnvs(): Map<string, string> {
    const config = goConfig();

    const envVars = new Map<string, string>();

    const testEnvConfig = config["testEnvVars"] || {};

    Object.keys(testEnvConfig).forEach((key) =>
        envVars.set(
            key,
            typeof testEnvConfig[key] === "string"
                ? resolvePath(testEnvConfig[key])
                : String(testEnvConfig[key]),
        ),
    );

    return envVars;
}

export function getTestFlags(): string[] {
    const config = goConfig();

    const testFlags: string[] =
        config["testFlags"] || config["buildFlags"] || [];

    return testFlags.map((x) => resolvePath(x)); // Use copy of the flags, dont pass the actual object from config
}
