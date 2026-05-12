import * as vscode from "vscode";
import { CodelensProvider } from "./CodelensProvider";
import {
    COMMAND_SUITE_DEBUG_TEST,
    COMMAND_SUITE_RUN,
    COMMAND_SUITE_RUN_TEST,
} from "./const";
import { Commands } from "./commands";

let disposables: vscode.Disposable[] = [];

export function activate(_context: vscode.ExtensionContext) {
    const codelensProvider = new CodelensProvider();

    vscode.languages.registerCodeLensProvider(
        { language: "go", scheme: "file" },
        codelensProvider,
    );

    vscode.commands.registerCommand(COMMAND_SUITE_RUN, Commands.suiteRun);
    vscode.commands.registerCommand(
        COMMAND_SUITE_RUN_TEST,
        Commands.suiteRunTest,
    );

    vscode.commands.registerCommand(
        COMMAND_SUITE_DEBUG_TEST,
        Commands.suiteDebugTest,
    );
}

export function deactivate() {
    for (const d of disposables) {
        d.dispose();
    }

    disposables = [];
}
