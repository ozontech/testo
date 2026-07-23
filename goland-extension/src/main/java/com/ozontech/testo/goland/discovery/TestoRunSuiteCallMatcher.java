package com.ozontech.testo.goland.discovery;

import com.goide.psi.GoArgumentList;
import com.goide.psi.GoBuiltinArgumentList;
import com.goide.psi.GoBuiltinCallExpr;
import com.goide.psi.GoCallExpr;
import com.goide.psi.GoCompositeLit;
import com.goide.psi.GoExpression;
import com.goide.psi.GoFile;
import com.goide.psi.GoImportSpec;
import com.goide.psi.GoParenthesesExpr;
import com.goide.psi.GoReferenceExpression;
import com.goide.psi.GoType;
import com.goide.psi.GoTypeReferenceExpression;
import com.goide.psi.GoTypeSpec;
import com.goide.psi.GoUnaryExpr;
import com.intellij.psi.PsiElement;
import com.intellij.psi.ResolveState;
import com.intellij.psi.util.PsiTreeUtil;
import org.jetbrains.annotations.Nullable;

final class TestoRunSuiteCallMatcher {
    private static final String TESTO_IMPORT_PATH = "github.com/ozontech/testo";
    private static final String TESTO_DEFAULT_IMPORT_NAME = "testo";

    private TestoRunSuiteCallMatcher() {
    }

    @Nullable
    static GoTypeSpec suiteFromRunSuiteCall(GoCallExpr call, GoFile goFile, TestoPackageIndex packageIndex) {
        if (!isTestoRunSuiteCall(call, goFile)) {
            return null;
        }

        GoArgumentList argumentList = call.getArgumentList();
        if (argumentList == null) {
            return null;
        }

        for (GoExpression argument : argumentList.getExpressionList()) {
            GoTypeSpec argumentType = suiteTypeFromExpression(argument);
            if (argumentType != null) {
                return argumentType;
            }

            GoTypeSpec fallbackType = packageIndex.findType(suiteNameFromExpression(argument));
            if (fallbackType != null) {
                return fallbackType;
            }
        }

        return null;
    }

    private static boolean isTestoRunSuiteCall(GoCallExpr call, GoFile goFile) {
        GoExpression expression = call.getExpression();
        if (!(expression instanceof GoReferenceExpression reference)) {
            return false;
        }

        PsiElement identifier = reference.getIdentifier();
        if (identifier == null || !"RunSuite".equals(identifier.getText())) {
            return false;
        }

        PsiElement qualifier = reference.getQualifier();
        if (qualifier == null) {
            return hasDotTestoImport(goFile);
        }

        String qualifierName = GoTypeNames.simpleTypeName(qualifier.getText());
        String importName = testoImportName(goFile);
        return qualifierName.equals(importName) || (TESTO_DEFAULT_IMPORT_NAME.equals(qualifierName) && hasTestoImport(goFile));
    }

    @Nullable
    private static GoTypeSpec suiteTypeFromExpression(GoExpression expression) {
        GoExpression unwrapped = unwrapExpression(expression);

        if (unwrapped instanceof GoCompositeLit compositeLit) {
            GoTypeReferenceExpression typeReferenceExpression = compositeLit.getTypeReferenceExpression();
            if (typeReferenceExpression != null) {
                PsiElement resolved = typeReferenceExpression.getReference().resolve();
                return resolved instanceof GoTypeSpec typeSpec ? typeSpec : null;
            }

            return resolveType(compositeLit.getType());
        }

        if (unwrapped instanceof GoBuiltinCallExpr builtinCall
                && builtinCall.getExpression() != null
                && "new".equals(builtinCall.getExpression().getText())) {
            GoBuiltinArgumentList argumentList = builtinCall.getBuiltinArgumentList();
            if (argumentList != null) {
                return resolveType(argumentList.getType());
            }
        }

        GoType goType = unwrapped.getGoType(ResolveState.initial());
        return resolveType(goType);
    }

    @Nullable
    private static GoTypeSpec resolveType(@Nullable GoType goType) {
        if (goType == null) {
            return null;
        }

        PsiElement resolved = goType.getTypeReferenceExpression() == null
                ? null
                : goType.getTypeReferenceExpression().getReference().resolve();
        return resolved instanceof GoTypeSpec typeSpec ? typeSpec : null;
    }

    private static GoExpression unwrapExpression(GoExpression expression) {
        GoExpression current = expression;
        while (true) {
            if (current instanceof GoParenthesesExpr parentheses && parentheses.getExpression() != null) {
                current = parentheses.getExpression();
                continue;
            }

            if (current instanceof GoUnaryExpr unary && isAddressOfExpression(unary) && unary.getExpression() != null) {
                current = unary.getExpression();
                continue;
            }

            return current;
        }
    }

    private static boolean isAddressOfExpression(GoUnaryExpr unary) {
        PsiElement operator = unary.getOperator();
        if (operator != null && "&".equals(operator.getText())) {
            return true;
        }

        return unary.getText().startsWith("&");
    }

    private static String suiteNameFromExpression(GoExpression expression) {
        GoExpression unwrapped = unwrapExpression(expression);

        if (unwrapped instanceof GoCompositeLit compositeLit) {
            GoTypeReferenceExpression typeReferenceExpression = compositeLit.getTypeReferenceExpression();
            if (typeReferenceExpression != null) {
                return GoTypeNames.simpleTypeName(typeReferenceExpression.getText());
            }

            return GoTypeNames.simpleTypeName(compositeLit.getType() == null ? "" : compositeLit.getType().getText());
        }

        if (unwrapped instanceof GoBuiltinCallExpr builtinCall
                && builtinCall.getExpression() != null
                && "new".equals(builtinCall.getExpression().getText())) {
            GoBuiltinArgumentList argumentList = builtinCall.getBuiltinArgumentList();
            return GoTypeNames.simpleTypeName(argumentList == null || argumentList.getType() == null ? "" : argumentList.getType().getText());
        }

        return GoTypeNames.simpleTypeName(unwrapped.getText());
    }

    @Nullable
    private static String testoImportName(GoFile goFile) {
        for (GoImportSpec importSpec : PsiTreeUtil.findChildrenOfType(goFile, GoImportSpec.class)) {
            if (!TESTO_IMPORT_PATH.equals(importSpec.getPath()) || importSpec.isDot()) {
                continue;
            }

            String alias = importSpec.getAlias();
            return alias == null || alias.isBlank() ? TESTO_DEFAULT_IMPORT_NAME : alias;
        }

        return null;
    }

    private static boolean hasDotTestoImport(GoFile goFile) {
        for (GoImportSpec importSpec : PsiTreeUtil.findChildrenOfType(goFile, GoImportSpec.class)) {
            if (TESTO_IMPORT_PATH.equals(importSpec.getPath()) && importSpec.isDot()) {
                return true;
            }
        }

        return false;
    }

    private static boolean hasTestoImport(GoFile goFile) {
        for (GoImportSpec importSpec : PsiTreeUtil.findChildrenOfType(goFile, GoImportSpec.class)) {
            if (TESTO_IMPORT_PATH.equals(importSpec.getPath())) {
                return true;
            }
        }

        return false;
    }
}
