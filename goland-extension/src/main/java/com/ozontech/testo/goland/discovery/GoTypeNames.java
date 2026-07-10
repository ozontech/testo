package com.ozontech.testo.goland.discovery;

import com.goide.psi.GoMethodDeclaration;
import com.goide.psi.GoType;
import com.goide.psi.GoTypeSpec;

import java.util.Objects;

final class GoTypeNames {
    private GoTypeNames() {
    }

    static boolean sameTypeSpec(GoTypeSpec left, GoTypeSpec right) {
        return left.isEquivalentTo(right) || Objects.equals(left.getQualifiedName(), right.getQualifiedName());
    }

    static boolean receiverMatchesSuiteName(GoMethodDeclaration method, GoTypeSpec suite) {
        GoType receiverType = method.getReceiverType();
        return receiverType != null && simpleTypeName(receiverType.getText()).equals(suite.getName());
    }

    static String simpleTypeName(String text) {
        if (text == null || text.isBlank()) {
            return "";
        }

        String result = text.trim();
        while (result.endsWith(".")) {
            result = result.substring(0, result.length() - 1).trim();
        }

        while (result.startsWith("*") || result.startsWith("&")) {
            result = result.substring(1).trim();
        }

        if (result.endsWith("{}")) {
            result = result.substring(0, result.length() - 2).trim();
        }

        int lastDot = result.lastIndexOf('.');
        if (lastDot >= 0) {
            result = result.substring(lastDot + 1);
        }

        int typeArgs = result.indexOf('[');
        if (typeArgs >= 0) {
            result = result.substring(0, typeArgs);
        }

        return result;
    }
}
