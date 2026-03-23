import { defineConfig } from "@trigger.dev/sdk";

export default defineConfig({
  project: process.env.TRIGGER_PROJECT_REF ?? "spiderweb",
  runtime: "node",
  logLevel: "log",
  dirs: ["./tasks"],
});
