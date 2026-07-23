package com.ozontech.testo.goland.discovery;

import com.goide.psi.GoFile;
import com.intellij.psi.PsiComment;

final class GoBuildTags {
    private GoBuildTags() {
    }

    static String buildFlags(GoFile goFile) {
        String buildFlags = goFile.getBuildFlags();
        if (buildFlags != null && !buildFlags.isBlank()) {
            return buildFlags;
        }

        PsiComment directive = goFile.getGoBuildDirectiveElement();
        if (directive == null) {
            return "";
        }

        return simpleGoBuildTag(directive.getText());
    }

    static String simpleGoBuildTag(String directiveText) {
        if (directiveText == null) {
            return "";
        }

        String expression = directiveText.replaceFirst("^\\s*//go:build\\s+", "").trim();
        return expression.matches("[A-Za-z0-9_][A-Za-z0-9_\\-.]*") ? expression : "";
    }
}
