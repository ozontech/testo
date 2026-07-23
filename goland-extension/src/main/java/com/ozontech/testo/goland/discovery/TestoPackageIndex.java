package com.ozontech.testo.goland.discovery;

import com.goide.psi.GoFile;
import com.goide.psi.GoMethodDeclaration;
import com.goide.psi.GoTypeSpec;
import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.psi.PsiElement;
import com.intellij.psi.PsiFile;
import com.intellij.psi.PsiManager;
import com.intellij.psi.util.CachedValueProvider;
import com.intellij.psi.util.CachedValuesManager;
import com.intellij.psi.util.PsiModificationTracker;
import com.intellij.psi.util.PsiTreeUtil;
import org.jetbrains.annotations.Nullable;

import java.util.Arrays;
import java.util.List;
import java.util.Objects;

final class TestoPackageIndex {
    private final List<GoFile> files;

    private TestoPackageIndex(List<GoFile> files) {
        this.files = files;
    }

    @Nullable
    static TestoPackageIndex forElement(PsiElement element) {
        if (element == null) {
            return null;
        }

        PsiFile containingFile = element.getContainingFile();
        if (!(containingFile instanceof GoFile goFile)) {
            return null;
        }

        return forFile(goFile);
    }

    static TestoPackageIndex forFile(GoFile goFile) {
        return CachedValuesManager.getCachedValue(goFile, () ->
                CachedValueProvider.Result.create(compute(goFile), PsiModificationTracker.MODIFICATION_COUNT)
        );
    }

    List<GoFile> files() {
        return files;
    }

    @Nullable
    GoTypeSpec findType(String name) {
        if (name == null || name.isBlank()) {
            return null;
        }

        for (GoFile goFile : files) {
            for (GoTypeSpec typeSpec : PsiTreeUtil.findChildrenOfType(goFile, GoTypeSpec.class)) {
                if (name.equals(typeSpec.getName())) {
                    return typeSpec;
                }
            }
        }

        return null;
    }

    boolean hasSuiteTest(GoTypeSpec suite) {
        for (GoFile goFile : files) {
            for (GoMethodDeclaration method : goFile.getMethods()) {
                if (!GoTestNames.isGoTestMethod(method.getName())) {
                    continue;
                }

                GoTypeSpec receiverType = method.resolveTypeSpec();
                if ((receiverType != null && GoTypeNames.sameTypeSpec(receiverType, suite))
                        || GoTypeNames.receiverMatchesSuiteName(method, suite)) {
                    return true;
                }
            }
        }

        return false;
    }

    private static TestoPackageIndex compute(GoFile goFile) {
        VirtualFile dir = goFile.getVirtualFile() == null ? null : goFile.getVirtualFile().getParent();
        if (dir == null) {
            return new TestoPackageIndex(List.of(goFile));
        }

        PsiManager psiManager = PsiManager.getInstance(goFile.getProject());
        List<GoFile> packageFiles = Arrays.stream(dir.getChildren())
                .filter(TestoPackageIndex::isGoFile)
                .map(psiManager::findFile)
                .filter(GoFile.class::isInstance)
                .map(GoFile.class::cast)
                .filter(candidate -> Objects.equals(candidate.getPackageName(), goFile.getPackageName()))
                .toList();
        return new TestoPackageIndex(packageFiles);
    }

    private static boolean isGoFile(VirtualFile file) {
        return file != null && file.getName().endsWith(".go");
    }
}
