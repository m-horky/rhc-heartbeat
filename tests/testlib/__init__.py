"""Support library for the rhc-heartbeat tmt integration tests.

The tests import this package after adding ``$TMT_TREE/tests`` to
``sys.path``. Every helper expects paths from the running tmt test, so the
package is intentionally not usable outside a tmt execution::

    from testlib.environment import TestContext

    with TestContext.from_tmt() as ctx:
        ctx.prepare()
        # run a scenario against ctx
"""
