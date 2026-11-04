"""unittest support for the tmt integration tests."""

from __future__ import annotations

import logging
import unittest

from .environment import TestContext


class TmtTestCase(unittest.TestCase):
    """Prepare a fresh TestContext for every test method.

    Subclasses define one ``test_*`` method per tmt test.
    """

    ctx: TestContext

    def setUp(self) -> None:
        """Create and prepare the context used by the test method."""
        self.ctx = TestContext.from_tmt()
        self.ctx.__enter__()
        self.addCleanup(self.ctx.__exit__, None, None, None)
        self.prepare_context()

        # Log the test's Gherkin statement
        if self._testMethodDoc:
            logging.info(self._testMethodDoc)

    def prepare_context(self) -> None:
        """Prepare the context; tiers can override this hook."""
        self.ctx.prepare()
