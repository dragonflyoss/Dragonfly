package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.file.Files;
import java.nio.file.Paths;

/**
 * @author lowzj
 */
class PreheatProcess {
    final private String output;
    final private Process process;

    PreheatProcess(String output, Process process) {
        assert process != null;
        this.output = output;
        this.process = process;
    }

    int exitValue() {
        return process.exitValue();
    }

    boolean isAlive() {
        try {
            process.exitValue();
            return false;
        } catch (IllegalThreadStateException ignored) {
            return true;
        }
    }

    void destroy() {
        try {
            Files.deleteIfExists(Paths.get(output));
        } catch (IOException ignored) {
        }
        if (isAlive()) {
            process.destroy();
        }
    }

    String getError() {
        return readOut(process.getErrorStream());
    }

    private String readOut(InputStream is) {
        StringBuilder sb = new StringBuilder();
        BufferedReader br = new BufferedReader(new InputStreamReader(is));
        String line;
        try {
            while ((line = br.readLine()) != null) {
                sb.append(line).append("\n");
            }
        } catch (IOException ignored) {
        }
        return sb.toString();
    }
}
