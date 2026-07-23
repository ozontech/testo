package com.ozontech.testo.goland.marker;

import com.intellij.execution.lineMarker.ExecutorAction;
import com.intellij.execution.lineMarker.RunLineMarkerContributor;
import com.intellij.icons.AllIcons;
import com.intellij.psi.PsiElement;
import com.ozontech.testo.goland.TestoPsiUtil;
import com.ozontech.testo.goland.TestoTarget;
import org.jetbrains.annotations.Nullable;

public final class TestoRunLineMarkerContributor extends RunLineMarkerContributor {
    @Nullable
    @Override
    public Info getInfo(PsiElement element) {
        TestoTarget target = TestoPsiUtil.targetAt(element);
        if (target == null) {
            return null;
        }

        String tooltip = target.isSuite()
                ? "Run Testo suite " + target.getSuiteName()
                : "Run Testo test " + target.getSuiteName() + "." + target.getTestName();

        return new Info(AllIcons.RunConfigurations.TestState.Run, ExecutorAction.getActions(0), ignored -> tooltip);
    }
}
