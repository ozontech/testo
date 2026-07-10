package com.ozontech.testo.goland.run;

import com.goide.execution.testing.GoTestRunConfiguration;
import com.goide.execution.testing.GoTestRunConfigurationType;
import com.intellij.execution.actions.ConfigurationContext;
import com.intellij.execution.actions.LazyRunConfigurationProducer;
import com.intellij.execution.configurations.ConfigurationFactory;
import com.intellij.openapi.util.Ref;
import com.intellij.psi.PsiElement;
import com.ozontech.testo.goland.TestoPsiUtil;
import com.ozontech.testo.goland.TestoTarget;
import org.jetbrains.annotations.NotNull;

public final class TestoRunConfigurationProducer extends LazyRunConfigurationProducer<GoTestRunConfiguration> {
    @Override
    public @NotNull ConfigurationFactory getConfigurationFactory() {
        return GoTestRunConfigurationType.getInstance().getFactory();
    }

    @Override
    protected boolean setupConfigurationFromContext(
            @NotNull GoTestRunConfiguration configuration,
            @NotNull ConfigurationContext context,
            @NotNull Ref<PsiElement> sourceElement
    ) {
        PsiElement element = context.getPsiLocation();
        TestoTarget target = TestoPsiUtil.targetAt(element);
        if (target == null) {
            return false;
        }

        TestoGoTestConfiguration.applyTo(configuration, target);
        sourceElement.set(element);
        return true;
    }

    @Override
    public boolean isConfigurationFromContext(
            @NotNull GoTestRunConfiguration configuration,
            @NotNull ConfigurationContext context
    ) {
        TestoTarget target = TestoPsiUtil.targetAt(context.getPsiLocation());
        return target != null && TestoGoTestConfiguration.isForTarget(configuration, target);
    }
}
