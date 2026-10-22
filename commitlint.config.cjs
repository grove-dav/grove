// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

module.exports = {
  extends: ["@commitlint/config-conventional"],
  // Dependabot bodies contain long release-note and compare URLs that exceed
  // body-max-line-length, and we can't change how it formats them.
  ignores: [(message) => /^Signed-off-by: dependabot\[bot\]/m.test(message)],
};
