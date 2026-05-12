import * as vscode from "vscode";
import { GO_TEST_FILE_SUFFIX, TESTO_EXTENSION_ID } from "./const";
import { DocumentScoped } from "./models";
import * as path from "node:path";
import * as os from "node:os";

export function testoConfig(): vscode.WorkspaceConfiguration {
  return vscode.workspace.getConfiguration(TESTO_EXTENSION_ID);
}

export function goConfig(): vscode.WorkspaceConfiguration {
  return vscode.workspace.getConfiguration("go");
}

export async function getDocumentSymbols(
  documentUri: vscode.Uri,
): Promise<vscode.DocumentSymbol[]> {
  const symbols = await vscode.commands.executeCommand<
    vscode.DocumentSymbol[] | undefined
  >("vscode.executeDocumentSymbolProvider", documentUri);

  if (!symbols) {
    return [];
  }

  return symbols;
}

export async function packageSymbols(
  dir: vscode.Uri,
  token: vscode.CancellationToken,
): Promise<DocumentScoped<vscode.DocumentSymbol>[]> {
  const files = await vscode.workspace.fs.readDirectory(dir);

  const symbolPromises: Promise<DocumentScoped<vscode.DocumentSymbol>[]>[] = [];

  for (const [fileName, fileType] of files) {
    if (token.isCancellationRequested) {
      break;
    }

    if (fileType !== vscode.FileType.File) {
      continue;
    }

    if (fileName.endsWith(".go")) {
      const fileUri = vscode.Uri.joinPath(dir, fileName);

      const doc = await vscode.workspace.openTextDocument(fileUri);

      const promise = getDocumentSymbols(fileUri).then((symbols) =>
        symbols.map((s) => new DocumentScoped(doc, s)),
      );

      symbolPromises.push(promise);
    }
  }

  const results = await Promise.all(symbolPromises);

  return results.flat();
}

const runTestSuiteRegex = /^\s*testo\.RunSuite\(/mu;
const testFuncRegex = /^Test$|^Test\P{Ll}.*|^Example$|^Example\P{Ll}.*/u;

export async function findSuiteCaller(
  documentUri: vscode.Uri,
  position: vscode.Position,
): Promise<DocumentScoped<vscode.DocumentSymbol> | undefined> {
  const locations = await vscode.commands.executeCommand<
    vscode.Location[] | undefined
  >("vscode.executeReferenceProvider", documentUri, position);

  if (!locations) {
    return;
  }

  for (const loc of locations) {
    const doc = await vscode.workspace.openTextDocument(loc.uri);

    if (!doc.fileName.endsWith(GO_TEST_FILE_SUFFIX)) {
      continue;
    }

    const [start, end] = [loc.range.start, loc.range.end];

    const text = doc.getText(
      new vscode.Range(start.with(start.line - 2, 0), end),
    );

    if (!runTestSuiteRegex.test(text)) {
      continue;
    }

    const symbols = await getDocumentSymbols(doc.uri);

    for (const sym of symbols) {
      if (sym.kind !== vscode.SymbolKind.Function) {
        continue;
      }

      if (!sym.range.contains(loc.range)) {
        continue;
      }

      if (!testFuncRegex.test(sym.name)) {
        continue;
      }

      return new DocumentScoped(doc, sym);
    }
  }

  return;
}

/**
 * Expands ~ to homedir in non-Windows platform and resolves
 * ${workspaceFolder}, ${workspaceRoot} and ${workspaceFolderBasename}
 */
export function resolvePath(
  inputPath: string,
  workspaceFolder?: string,
): string {
  if (!inputPath || !inputPath.trim()) {
    return inputPath;
  }

  if (!workspaceFolder && vscode.workspace.workspaceFolders) {
    workspaceFolder = getWorkspaceFolderPath(
      vscode.window.activeTextEditor &&
        vscode.window.activeTextEditor.document.uri,
    );
  }

  if (workspaceFolder) {
    inputPath = inputPath.replace(
      /\${workspaceFolder}|\${workspaceRoot}/g,
      workspaceFolder,
    );
    inputPath = inputPath.replace(
      /\${workspaceFolderBasename}/g,
      path.basename(workspaceFolder),
    );
  }

  return resolveHomeDir(inputPath);
}

export function getWorkspaceFolderPath(
  fileUri?: vscode.Uri,
): string | undefined {
  if (fileUri) {
    const workspace = vscode.workspace.getWorkspaceFolder(fileUri);
    if (workspace) {
      return fixDriveCasingInWindows(workspace.uri.fsPath);
    }
  }

  // fall back to the first workspace
  const folders = vscode.workspace.workspaceFolders;
  if (folders && folders.length) {
    return fixDriveCasingInWindows(folders[0].uri.fsPath);
  }
  return undefined;
}

// Workaround for issue in https://github.com/Microsoft/vscode/issues/9448#issuecomment-244804026
export function fixDriveCasingInWindows(pathToFix: string): string {
  return process.platform === "win32" && pathToFix
    ? pathToFix.substring(0, 1).toUpperCase() + pathToFix.substring(1)
    : pathToFix;
}

/**
 * Exapnds ~ to homedir in non-Windows platform
 */
export function resolveHomeDir(inputPath: string): string {
  if (!inputPath || !inputPath.trim()) {
    return inputPath;
  }

  return inputPath.startsWith("~")
    ? path.join(os.homedir(), inputPath.substring(1))
    : inputPath;
}
