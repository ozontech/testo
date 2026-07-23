package com.ozontech.testo.goland.run;

import com.goide.execution.GoBuildingRunConfiguration;
import com.goide.execution.testing.GoTestRunConfiguration;
import com.goide.execution.testing.frameworks.gotest.GotestFramework;
import com.ozontech.testo.goland.TestoTarget;

import java.util.Objects;

public final class TestoGoTestConfiguration {
    private TestoGoTestConfiguration() {
    }

    public static void applyTo(GoTestRunConfiguration configuration, TestoTarget target) {
        configuration.setName(target.configurationName());
        if (target.getPackageImportPath() == null || target.getPackageImportPath().isBlank()) {
            configuration.setKind(GoBuildingRunConfiguration.Kind.DIRECTORY);
            configuration.setDirectoryPath(target.getPackageDir());
        } else {
            configuration.setKind(GoBuildingRunConfiguration.Kind.PACKAGE);
            configuration.setPackage(target.getPackageImportPath());
        }
        configuration.setDirectoryPath(target.getPackageDir());
        configuration.setWorkingDirectory(target.getPackageDir());
        configuration.setGoToolParams(goToolParamsFor(target));
        configuration.setPattern(patternFor(target));
        configuration.setParams(paramsFor(target));
        configuration.setTestFramework(GotestFramework.INSTANCE);
    }

    public static boolean isForTarget(GoTestRunConfiguration configuration, TestoTarget target) {
        GoBuildingRunConfiguration.Kind expectedKind = target.getPackageImportPath() == null || target.getPackageImportPath().isBlank()
                ? GoBuildingRunConfiguration.Kind.DIRECTORY
                : GoBuildingRunConfiguration.Kind.PACKAGE;

        return configuration.getKind() == expectedKind
                && Objects.equals(configuration.getDirectoryPath(), target.getPackageDir())
                && Objects.equals(configuration.getPackage(), target.getPackageImportPath())
                && Objects.equals(configuration.getGoToolParams(), goToolParamsFor(target))
                && Objects.equals(configuration.getPattern(), patternFor(target))
                && Objects.equals(configuration.getParams(), paramsFor(target));
    }

    public static String patternFor(TestoTarget target) {
        return "^" + target.getSuiteCallerTest() + "$/^" + target.getSuiteName() + "$";
    }

    public static String paramsFor(TestoTarget target) {
        if (target.isSuite()) {
            return "";
        }

        return "-testo.m ^" + target.getTestName() + "$";
    }

    public static String goToolParamsFor(TestoTarget target) {
        if (target.getBuildFlags() == null || target.getBuildFlags().isBlank()) {
            return "";
        }

        return "-tags " + target.getBuildFlags();
    }
}
