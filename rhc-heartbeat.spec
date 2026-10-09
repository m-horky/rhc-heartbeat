%bcond check 1
# go-vendor-tools are only available in Fedora.
%bcond_with go_vendor_tools

%global goipath github.com/m-horky/rhc-heartbeat
Version:        0.2.0
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

%install
%if %{with go_vendor_tools}
%go_vendor_license_install -c %{S:2}
%endif

# Binaries
install -m 0755 -vd                     %{buildroot}%{_bindir}
install -m 0755 -vp %{gobuilddir}/bin/* %{buildroot}%{_bindir}/

# Configuration
install -m 0755 -vd                     %{buildroot}%{_prefix}/lib/rhc/
install -m 0755 -vd                     %{buildroot}%{_sysconfdir}/rhc/
install -m 0755 -vd                     %{buildroot}%{_sysconfdir}/rhc/rhc-heartbeat.conf.d/

# Systemd units
install -m 0755 -vd                     %{buildroot}%{_unitdir}
install -m 0644 -vp data/systemd/*.service data/systemd/*.timer %{buildroot}%{_unitdir}/

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

%files %{extra_files}
# Binaries
%{_bindir}/rhc-heartbeat
# Runtime and configuration directories
%dir %{_prefix}/lib/rhc/
%dir %{_sysconfdir}/rhc/
%dir %{_sysconfdir}/rhc/rhc-heartbeat.conf.d/
# Systemd units
%{_unitdir}/rhc-heartbeat.timer
%{_unitdir}/rhc-heartbeat.service
%{_unitdir}/rhc-heartbeat-sleep.service
%{_unitdir}/rhc-heartbeat-off.service
# Documentation
%license LICENSE
%doc README.md SECURITY.md

%changelog
%autochangelog
