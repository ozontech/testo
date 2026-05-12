import * as vscode from "vscode";
import {
    COMMAND_SUITE_DEBUG_TEST,
    COMMAND_SUITE_RUN,
    COMMAND_SUITE_RUN_TEST,
} from "./const";
import { packageSymbols } from "./utils";
import { DocumentScoped, Suite, SuiteTest } from "./models";

/**
 * CodelensProvider
 */
export class CodelensProvider implements vscode.CodeLensProvider {
    private _onDidChangeCodeLenses: vscode.EventEmitter<void> =
        new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses: vscode.Event<void> =
        this._onDidChangeCodeLenses.event;

    constructor() {
        vscode.workspace.onDidChangeConfiguration((_) => {
            this._onDidChangeCodeLenses.fire();
        });
    }

    public async provideCodeLenses(
        document: vscode.TextDocument,
        token: vscode.CancellationToken,
    ): Promise<vscode.CodeLens[]> {
        const symbols = await packageSymbols(
            vscode.Uri.joinPath(document.uri, ".."),
            token,
        );

        const testsBySuite = new Map<string, SuiteTest<string>[]>();
        const symbolsByName = new Map<
            string,
            DocumentScoped<vscode.DocumentSymbol>
        >();

        for (const sym of symbols) {
            symbolsByName.set(sym.value.name, sym);

            const test = SuiteTest.fromSymbol(sym.document, sym.value);

            if (!test) {
                continue;
            }

            const tests = testsBySuite.get(test.suite) ?? [];

            tests.push(test);

            testsBySuite.set(test.suite, tests);
        }

        const codeLenses: vscode.CodeLens[] = [];

        for (const [suite, tests] of testsBySuite.entries()) {
            const suiteSym = symbolsByName.get(suite);

            if (!suiteSym || !tests) {
                continue;
            }

            if (suiteSym.document === document) {
                codeLenses.push(
                    new vscode.CodeLens(suiteSym.value.range, {
                        title: "run testo suite",
                        command: COMMAND_SUITE_RUN,
                        arguments: [
                            new Suite(suiteSym.document, suiteSym.value),
                        ],
                    }),
                );
            }

            for (const test of tests) {
                if (test.doc !== document) {
                    continue;
                }

                codeLenses.push(
                    new vscode.CodeLens(test.range, {
                        title: "run testo test",
                        command: COMMAND_SUITE_RUN_TEST,
                        arguments: [
                            new SuiteTest(
                                document,
                                suiteSym,
                                test.name,
                                test.range,
                            ),
                        ],
                    }),
                    new vscode.CodeLens(test.range, {
                        title: "debug testo test",
                        command: COMMAND_SUITE_DEBUG_TEST,
                        arguments: [
                            new SuiteTest(
                                document,
                                suiteSym,
                                test.name,
                                test.range,
                            ),
                        ],
                    }),
                );
            }
        }

        return codeLenses;
    }
}
