package com.ozontech.testo.goland;

import com.intellij.psi.PsiElement;
import com.ozontech.testo.goland.discovery.TestoTargetResolver;
import org.jetbrains.annotations.Nullable;

public final class TestoPsiUtil {
    private TestoPsiUtil() {
    }

    @Nullable
    public static TestoTarget targetAt(PsiElement element) {
        return TestoTargetResolver.targetAt(element);
    }
}
