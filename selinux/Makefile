# Build the rhc_heartbeat SELinux policy module.
#
# Requires the selinux-policy-devel package, which provides the Makefile
# referenced below along with the reference policy headers.

MODULE := rhc_heartbeat
DEVEL_MAKEFILE := /usr/share/selinux/devel/Makefile

.PHONY: all install clean

all: $(MODULE).pp

$(MODULE).pp: $(MODULE).te $(MODULE).if $(MODULE).fc
	$(MAKE) -f $(DEVEL_MAKEFILE) $@

# Load the module into the running policy. Useful for iterating on a
# test machine without rebuilding the RPM.
install: all
	/usr/sbin/semodule -i $(MODULE).pp

clean:
	rm -f $(MODULE).pp
