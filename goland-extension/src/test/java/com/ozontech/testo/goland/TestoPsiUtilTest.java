package com.ozontech.testo.goland;

import com.intellij.psi.PsiElement;
import com.intellij.testFramework.fixtures.BasePlatformTestCase;

public final class TestoPsiUtilTest extends BasePlatformTestCase {
    public void testFindsSuiteTestWithAliasedTestoImport() {
        myFixture.configureByText("suite_test.go", """
                //go:build example

                package sample

                import (
                    "testing"
                    tst "github.com/ozontech/testo"
                )

                type UserSuite struct{}

                func TestUserSuite(t *testing.T) {
                    tst.RunSuite(t, &UserSuite{})
                }

                func (s *UserSuite) <caret>TestCreatesUser() {}
                """);

        TestoTarget target = targetAtCaret();

        assertNotNull(target);
        assertEquals("TestUserSuite", target.getSuiteCallerTest());
        assertEquals("UserSuite", target.getSuiteName());
        assertEquals("TestCreatesUser", target.getTestName());
        assertEquals("example", target.getBuildFlags());
    }

    public void testFindsSuiteRunSuiteCallWithDotTestoImport() {
        myFixture.configureByText("suite_test.go", """
                package sample

                import (
                    "testing"
                    . "github.com/ozontech/testo"
                )

                type PaymentSuite struct{}

                func TestPaymentSuite(t *testing.T) {
                    Run<caret>Suite(t, new(PaymentSuite))
                }

                func (s *PaymentSuite) TestChargesCard() {}
                """);

        TestoTarget target = targetAtCaret();

        assertNotNull(target);
        assertEquals("TestPaymentSuite", target.getSuiteCallerTest());
        assertEquals("PaymentSuite", target.getSuiteName());
        assertNull(target.getTestName());
    }

    public void testFindsRunSuiteCallWhenSuiteDeclarationIsOutsideTestFile() {
        myFixture.addFileToProject("suite.go", """
                package sample

                type InventorySuite struct{}
                """);

        myFixture.configureByText("suite_test.go", """
                package sample

                import (
                    "testing"
                    "github.com/ozontech/testo"
                )

                func TestInventorySuite(t *testing.T) {
                    testo.Run<caret>Suite(t, InventorySuite{})
                }

                func (s *InventorySuite) TestListsItems() {}
                """);

        TestoTarget target = targetAtCaret();

        assertNotNull(target);
        assertEquals("TestInventorySuite", target.getSuiteCallerTest());
        assertEquals("InventorySuite", target.getSuiteName());
        assertNull(target.getTestName());
    }

    public void testIgnoresSuiteCallerFunctionToAvoidGoLandGutterConflicts() {
        myFixture.configureByText("suite_test.go", """
                package sample

                import (
                    "testing"
                    "github.com/ozontech/testo"
                )

                type InventorySuite struct{}

                func <caret>TestInventorySuite(t *testing.T) {
                    testo.RunSuite(t, InventorySuite{})
                }

                func (s *InventorySuite) TestListsItems() {}
                """);

        assertNull(targetAtCaret());
    }

    public void testIgnoresSuiteDeclarationToAvoidGoLandGutterConflicts() {
        myFixture.addFileToProject("suite_test.go", """
                package sample

                import (
                    "testing"
                    "github.com/ozontech/testo"
                )

                func TestInventorySuite(t *testing.T) {
                    testo.RunSuite(t, InventorySuite{})
                }

                func (s *InventorySuite) TestListsItems() {}
                """);

        myFixture.configureByText("suite.go", """
                package sample

                type <caret>InventorySuite struct{}
                """);

        assertNull(targetAtCaret());
    }

    public void testIgnoresNonTestMethods() {
        myFixture.configureByText("suite_test.go", """
                package sample

                import (
                    "testing"
                    "github.com/ozontech/testo"
                )

                type UserSuite struct{}

                func TestUserSuite(t *testing.T) {
                    testo.RunSuite(t, &UserSuite{})
                }

                func (s *UserSuite) <caret>Helper() {}
                """);

        assertNull(targetAtCaret());
    }

    private TestoTarget targetAtCaret() {
        PsiElement element = myFixture.getFile().findElementAt(myFixture.getCaretOffset());
        return TestoPsiUtil.targetAt(element);
    }
}
