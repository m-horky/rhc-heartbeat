# Unit tests during build.
%bcond check 1
# SELinux policy module, built from selinux/.
%bcond selinux 1
# go-vendor-tools are only available in Fedora.
%bcond_with go_vendor_tools

%global goipath github.com/m-horky/rhc-heartbeat
Version:        0.3.1
%gometa -L -f

Name:           rhc-heartbeat
Release:        %autorelease
Summary:        Manage system heartbeats
License:        Apache-2.0 AND BSD-3-Clause AND GPL-3.0-only AND MIT AND MPL-2.0
URL:            %{gourl}
Source0:        %{gosource}
Source1:        %{archivename}-vendor.tar.bz2
Source2:        go-vendor-tools.toml

Requires:       subscription-manager

BuildRequires:  systemd-rpm-macros
%if %{with selinux}
BuildRequires:  selinux-policy-devel
%endif
%if %{with go_vendor_tools}
BuildRequires:  go-vendor-tools
BuildRequires:  askalono-cli
%endif

%description
rhc-heartbeat collects system profile and monotonic time.
It uploads the heartbeat to an Prometheus Remote Write endpoint.

%prep
# Unpack Source0 and set up the go build directory. Since -k is not passed in,
# the vendor/ directory from tarball is explicitly deleted.
%goprep -p1
# Unpack Source1 into the build tree, providing the vendor/ directory.
%setup -q -T -D -a1 -n %{name}-%{version}
# Apply patches, if present
#autopatch -p1

%generate_buildrequires
%if %{with go_vendor_tools}
%go_vendor_license_buildrequires -c %{S:2}
%endif

%build
export GO_LDFLAGS="-X %{goipath}/pkg/version.Version=%{version}"
for cmd in cmd/* ; do
  %gobuild -o %{gobuilddir}/bin/$(basename $cmd) %{goipath}/$cmd
done

# SELinux policy module
%if %{with selinux}
%make_build -C selinux
%endif

%install
%if %{with go_vendor_tools}
%go_vendor_license_install -c %{S:2}
%endif

# Main command and subcommands
install -m 0755 -vd                     %{buildroot}%{_libexecdir}/rhc/
install -m 0755 -vp %{gobuilddir}/bin/rhc-heartbeat %{buildroot}%{_libexecdir}/rhc/heartbeat
# SELinux transition wrapper for subscription-manager
install -m 0755 -vp data/wrapper/heartbeat-rhsm %{buildroot}%{_libexecdir}/rhc/heartbeat-rhsm

# Configuration
install -m 0755 -vd                     %{buildroot}%{_prefix}/lib/rhc/
install -m 0755 -vd                     %{buildroot}%{_sysconfdir}/rhc/
install -m 0755 -vd                     %{buildroot}%{_sysconfdir}/rhc/rhc-heartbeat.conf.d/

# Systemd units
install -m 0755 -vd                     %{buildroot}%{_unitdir}
install -m 0644 -vp data/systemd/*.service data/systemd/*.timer %{buildroot}%{_unitdir}/

# SELinux policy module
%if %{with selinux}
install -m 0755 -vd                     %{buildroot}%{_datadir}/selinux/packages
install -m 0644 -vp selinux/rhc_heartbeat.pp %{buildroot}%{_datadir}/selinux/packages/
%endif

%check
%if %{with go_vendor_tools}
%go_vendor_license_check -c %{S:2}
%endif

%if %{with check}
# Trigger unit tests, unless explicitly disabled.
%gocheck
%endif

%if %{with go_vendor_tools}
%global extra_files -f %{go_vendor_license_filelist}
%else
%global extra_files %{nil}
%endif

%if %{with selinux}
%package selinux
Summary:        SELinux policy module for rhc-heartbeat
BuildArch:      noarch
Requires:       %{name} = %{version}-%{release}
Requires(post): selinux-policy-base
Requires(post): policycoreutils
Requires(postun): policycoreutils
%description selinux
SELinux policy module confining the rhc-heartbeat daemon.
%endif

%files %{extra_files}
# Binaries
%dir %{_libexecdir}/rhc/
%{_libexecdir}/rhc/heartbeat
%{_libexecdir}/rhc/heartbeat-rhsm
# Runtime and configuration directories
%dir %{_prefix}/lib/rhc/
%dir %{_sysconfdir}/rhc/
%dir %{_sysconfdir}/rhc/rhc-heartbeat.conf.d/
# Systemd units
%{_unitdir}/rhc-heartbeat.timer
%{_unitdir}/rhc-heartbeat.service
%{_unitdir}/rhc-heartbeat-sleep.service
%{_unitdir}/rhc-heartbeat-off.service
%{_unitdir}/rhc-heartbeat-on.service
# Documentation
%license LICENSE
%doc README.md SECURITY.md

%if %{with selinux}
%files selinux
%attr(0644,root,root) %{_datadir}/selinux/packages/rhc_heartbeat.pp
%endif

%if %{with selinux}
%pre selinux
# Snapshot the current file contexts so that %posttrans can relabel only
# the paths this module changes.
%selinux_relabel_pre

%post selinux
%selinux_modules_install %{_datadir}/selinux/packages/rhc_heartbeat.pp
# Heal stale labels: %selinux_relabel_post only relabels paths whose rules
# changed during this transaction, so files that were mislabeled before the
# module existed (for example while a module install had failed) are fixed
# here explicitly. restorecon is idempotent and applies all current rules.
if /usr/sbin/selinuxenabled; then
  /usr/sbin/restorecon -R %{_libexecdir}/rhc || :
  /usr/sbin/restorecon -R %{_unitdir}/rhc-heartbeat* || :
fi

%postun selinux
if [ $1 -eq 0 ]; then
  %selinux_modules_uninstall rhc_heartbeat
  # Reset the labels of our files to the base policy defaults, now that
  # the module and its file contexts are gone.
  if /usr/sbin/selinuxenabled; then
    /usr/sbin/restorecon -R %{_libexecdir}/rhc || :
    /usr/sbin/restorecon -R %{_unitdir}/rhc-heartbeat* || :
  fi
fi

%posttrans selinux
# Relabel files affected by the module's file contexts: the binary and
# units installed by the main package, and any runtime state (cache, lock)
# that already exists. Runs after the module is loaded in %post.
%selinux_relabel_post
%endif

%changelog
%autochangelog
