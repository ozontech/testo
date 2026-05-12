import * as vscode from "vscode";

export class DocumentScoped<T> {
  public readonly document: vscode.TextDocument;
  public readonly value: T;

  constructor(document: vscode.TextDocument, value: T) {
    this.document = document;
    this.value = value;
  }
}

export class Suite {
  public readonly doc: vscode.TextDocument;
  public readonly symbol: vscode.DocumentSymbol;

  constructor(doc: vscode.TextDocument, symbol: vscode.DocumentSymbol) {
    this.doc = doc;
    this.symbol = symbol;
  }
}

export class SuiteTest<Suite> {
  public readonly suite: Suite;
  public readonly name: string;
  public readonly doc: vscode.TextDocument;
  public readonly range: vscode.Range;

  private static readonly testMethodRegex =
    /^\(\*?(?<suite>[^)]+)\)\.(?<test>Test|Test\P{Ll}.*)$/u;

  constructor(
    doc: vscode.TextDocument,
    suite: Suite,
    name: string,
    range: vscode.Range,
  ) {
    this.doc = doc;
    this.suite = suite;
    this.name = name;
    this.range = range;
  }

  public static fromSymbol(
    doc: vscode.TextDocument,
    sym: vscode.DocumentSymbol,
  ): SuiteTest<string> | undefined {
    if (sym.kind !== vscode.SymbolKind.Method) {
      return;
    }

    const matches = SuiteTest.testMethodRegex.exec(sym.name)?.groups;

    if (!matches) {
      return;
    }

    const suite = matches["suite"];
    const test = matches["test"];

    return new SuiteTest(doc, suite, test, sym.range);
  }
}
