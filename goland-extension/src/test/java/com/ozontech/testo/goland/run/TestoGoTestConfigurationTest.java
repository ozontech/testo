package com.ozontech.testo.goland.run;

import com.ozontech.testo.goland.TestoTarget;
import junit.framework.TestCase;

public final class TestoGoTestConfigurationTest extends TestCase {
    public void testBuildsSuitePatternWithoutExtraParams() {
        TestoTarget target = new TestoTarget("/repo/pkg", "TestUserSuite", "UserSuite", null);

        assertEquals("^TestUserSuite$/^UserSuite$", TestoGoTestConfiguration.patternFor(target));
        assertEquals("", TestoGoTestConfiguration.paramsFor(target));
    }

    public void testBuildsSuiteTestPatternAndTestoMethodFilter() {
        TestoTarget target = new TestoTarget("/repo/pkg", "TestUserSuite", "UserSuite", "TestCreatesUser");

        assertEquals("^TestUserSuite$/^UserSuite$", TestoGoTestConfiguration.patternFor(target));
        assertEquals("-testo.m ^TestCreatesUser$", TestoGoTestConfiguration.paramsFor(target));
    }

    public void testBuildsGoToolParamsForBuildFlags() {
        TestoTarget target = new TestoTarget(
                "/repo/examples/01_suite",
                "github.com/ozontech/testo/examples/01_suite",
                "example",
                "Test",
                "Suite",
                "TestMath"
        );

        assertEquals("-tags example", TestoGoTestConfiguration.goToolParamsFor(target));
    }
}
