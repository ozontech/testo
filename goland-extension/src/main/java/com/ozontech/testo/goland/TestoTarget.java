package com.ozontech.testo.goland;

import java.util.Objects;

public final class TestoTarget {
    private final String packageDir;
    private final String packageImportPath;
    private final String buildFlags;
    private final String suiteCallerTest;
    private final String suiteName;
    private final String testName;

    public TestoTarget(String packageDir, String suiteCallerTest, String suiteName, String testName) {
        this(packageDir, "", "", suiteCallerTest, suiteName, testName);
    }

    public TestoTarget(String packageDir, String packageImportPath, String buildFlags, String suiteCallerTest, String suiteName, String testName) {
        this.packageDir = packageDir;
        this.packageImportPath = packageImportPath;
        this.buildFlags = buildFlags;
        this.suiteCallerTest = suiteCallerTest;
        this.suiteName = suiteName;
        this.testName = testName;
    }

    public String getPackageDir() {
        return packageDir;
    }

    public String getPackageImportPath() {
        return packageImportPath;
    }

    public String getBuildFlags() {
        return buildFlags;
    }

    public String getSuiteCallerTest() {
        return suiteCallerTest;
    }

    public String getSuiteName() {
        return suiteName;
    }

    public String getTestName() {
        return testName;
    }

    public boolean isSuite() {
        return testName == null || testName.isBlank();
    }

    public String configurationName() {
        if (isSuite()) {
            return "Testo " + suiteName;
        }

        return "Testo " + suiteName + "." + testName;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (!(o instanceof TestoTarget that)) {
            return false;
        }
        return Objects.equals(packageDir, that.packageDir)
                && Objects.equals(packageImportPath, that.packageImportPath)
                && Objects.equals(buildFlags, that.buildFlags)
                && Objects.equals(suiteCallerTest, that.suiteCallerTest)
                && Objects.equals(suiteName, that.suiteName)
                && Objects.equals(testName, that.testName);
    }

    @Override
    public int hashCode() {
        return Objects.hash(packageDir, packageImportPath, buildFlags, suiteCallerTest, suiteName, testName);
    }
}
