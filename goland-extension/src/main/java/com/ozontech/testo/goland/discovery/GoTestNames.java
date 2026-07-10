package com.ozontech.testo.goland.discovery;

final class GoTestNames {
    private GoTestNames() {
    }

    static boolean isGoTestCaller(String name) {
        return isGoTestName(name, "Test") || isGoTestName(name, "Example");
    }

    static boolean isGoTestMethod(String name) {
        return isGoTestName(name, "Test");
    }

    private static boolean isGoTestName(String name, String prefix) {
        if (name == null) {
            return false;
        }

        if (name.equals(prefix)) {
            return true;
        }

        if (!name.startsWith(prefix) || name.length() == prefix.length()) {
            return false;
        }

        return !Character.isLowerCase(name.charAt(prefix.length()));
    }
}
