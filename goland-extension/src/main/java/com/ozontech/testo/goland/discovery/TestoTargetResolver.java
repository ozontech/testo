package com.ozontech.testo.goland.discovery;

import com.goide.psi.GoCallExpr;
import com.goide.psi.GoFile;
import com.goide.psi.GoFunctionDeclaration;
import com.goide.psi.GoMethodDeclaration;
import com.goide.psi.GoReferenceExpression;
import com.goide.psi.GoTypeSpec;
import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.psi.PsiElement;
import com.intellij.psi.PsiFile;
import com.intellij.psi.util.PsiTreeUtil;
import com.ozontech.testo.goland.TestoTarget;
import org.jetbrains.annotations.Nullable;

import java.util.Collection;

public final class TestoTargetResolver {
    private TestoTargetResolver() {
    }

    @Nullable
    public static TestoTarget targetAt(PsiElement element) {
        if (element == null) {
            return null;
        }

        TestoPackageIndex packageIndex = TestoPackageIndex.forElement(element);
        if (packageIndex == null) {
            return null;
        }

        TestoTarget runSuiteTarget = runSuiteCallTarget(element, packageIndex);
        if (runSuiteTarget != null) {
            return runSuiteTarget;
        }

        GoMethodDeclaration method = PsiTreeUtil.getParentOfType(element, GoMethodDeclaration.class, false);
        if (method != null && isDeclarationIdentifier(element, method.getIdentifier())) {
            return methodTarget(method, packageIndex);
        }

        return null;
    }

    @Nullable
    private static TestoTarget methodTarget(GoMethodDeclaration method, TestoPackageIndex packageIndex) {
        String testName = method.getName();
        if (!GoTestNames.isGoTestMethod(testName)) {
            return null;
        }

        GoTypeSpec suite = method.resolveTypeSpec();
        if (suite == null) {
            String receiverName = GoTypeNames.simpleTypeName(method.getReceiverType() == null ? "" : method.getReceiverType().getText());
            suite = packageIndex.findType(receiverName);
        }
        if (suite == null || suite.getName() == null) {
            return null;
        }

        SuiteCaller caller = findSuiteCaller(suite, packageIndex);
        return caller == null ? null : targetFor(caller, suite, testName);
    }

    @Nullable
    private static TestoTarget runSuiteCallTarget(PsiElement element, TestoPackageIndex packageIndex) {
        GoReferenceExpression reference = PsiTreeUtil.getParentOfType(element, GoReferenceExpression.class, false);
        if (reference == null || !isDeclarationIdentifier(element, reference.getIdentifier())) {
            return null;
        }

        GoCallExpr call = PsiTreeUtil.getParentOfType(reference, GoCallExpr.class, false);
        if (call == null || call.getExpression() != reference) {
            return null;
        }

        GoFunctionDeclaration function = PsiTreeUtil.getParentOfType(call, GoFunctionDeclaration.class);
        if (function == null || !GoTestNames.isGoTestCaller(function.getName())) {
            return null;
        }

        PsiFile containingFile = function.getContainingFile();
        if (!(containingFile instanceof GoFile goFile) || !isGoTestFile(goFile)) {
            return null;
        }

        GoTypeSpec suite = TestoRunSuiteCallMatcher.suiteFromRunSuiteCall(call, goFile, packageIndex);
        if (suite == null || suite.getName() == null || !packageIndex.hasSuiteTest(suite)) {
            return null;
        }

        SuiteCaller caller = suiteCaller(function, goFile);
        return caller == null ? null : targetFor(caller, suite, null);
    }

    @Nullable
    private static SuiteCaller findSuiteCaller(GoTypeSpec suite, TestoPackageIndex packageIndex) {
        for (GoFile goFile : packageIndex.files()) {
            if (!isGoTestFile(goFile)) {
                continue;
            }

            for (GoFunctionDeclaration function : goFile.getFunctions()) {
                if (!GoTestNames.isGoTestCaller(function.getName())) {
                    continue;
                }

                Collection<GoCallExpr> calls = PsiTreeUtil.findChildrenOfType(function.getBlock(), GoCallExpr.class);
                for (GoCallExpr call : calls) {
                    GoTypeSpec callSuite = TestoRunSuiteCallMatcher.suiteFromRunSuiteCall(call, goFile, packageIndex);
                    if (callSuite != null && GoTypeNames.sameTypeSpec(callSuite, suite)) {
                        return suiteCaller(function, goFile);
                    }
                }
            }
        }

        return null;
    }

    @Nullable
    private static SuiteCaller suiteCaller(GoFunctionDeclaration function, GoFile goFile) {
        VirtualFile dir = goFile.getVirtualFile() == null ? null : goFile.getVirtualFile().getParent();
        if (dir == null) {
            return null;
        }

        return new SuiteCaller(dir.getPath(), goFile.getImportPath(false), GoBuildTags.buildFlags(goFile), function.getName());
    }

    private static TestoTarget targetFor(SuiteCaller caller, GoTypeSpec suite, @Nullable String testName) {
        return new TestoTarget(caller.packageDir(), caller.packageImportPath(), caller.buildFlags(), caller.name(), suite.getName(), testName);
    }

    private static boolean isDeclarationIdentifier(PsiElement element, @Nullable PsiElement identifier) {
        return identifier != null
                && (element == identifier || element.getParent() == identifier || identifier.getTextRange().contains(element.getTextRange()));
    }

    private static boolean isGoTestFile(GoFile file) {
        VirtualFile virtualFile = file.getVirtualFile();
        return virtualFile != null && virtualFile.getName().endsWith("_test.go");
    }

    private record SuiteCaller(String packageDir, String packageImportPath, String buildFlags, String name) {
    }
}
