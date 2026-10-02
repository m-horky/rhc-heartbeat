%bcond check 1

%global goipath github.com/m-horky/rhc-heartbeat
Version:        0.1.0
%gometa -L -f

Name:           rhc-heartbeat
Release:        %autorelease
Summary:        Manage system heartbeats
License:        Apache-2.0 AND BSD-3-Clause AND GPL-3.0-only AND MIT AND MPL-2.0
URL:            %{gourl}
Source0:        %{gosource}
Source1:        %{archivename}-vendor.tar.bz2
Source2:        go-vendor-tools.toml

BuildRequires:  systemd-rpm-macros
%if 0%{?fedora}
# These build-time dependencies only exist for Fedora.
BuildRequires:  go-vendor-tools
BuildRequires:  askalono-cli
%endif

%description
rhc-heartbeat collects system uptime, wall-clock time, and time-synchronization status, then uploads the heartbeat to an OpenTelemetry endpoint.

%prep
# Unpack Source0 and set up the go build directory. Since -k is not passed in,
# the vendor/ directory from tarball is explicitly deleted.
%goprep -p1
# Unpack Source1 into the build tree, providing the vendor/ directory.
%setup -q -T -D -a1 -n %{name}-%{version}
# Apply patches, if present
%autopatch -p1

%generate_buildrequires
%if 0%{?fedora}
# Generate data for the licence check go-vendor-tools provides.
%go_vendor_license_buildrequires -c %{S:2}
%endif

%build
export GO_LDFLAGS="-X %{goipath}/pkg/version.Version=%{version}"
for cmd in cmd/* ; do
  %gobuild -o %{gobuilddir}/bin/$(basename $cmd) %{goipath}/$cmd
done

%install
%if 0%{?fedora}
# Only go-vendor-tools are capable of collecting and packaging licenses for our dependencies.
%go_vendor_license_install -c %{S:2}
%endif

# Binaries
install -m 0755 -vd                     %{buildroot}%{_bindir}
install -m 0755 -vp %{gobuilddir}/bin/* %{buildroot}%{_bindir}/

# Configuration
install -m 0755 -vd                     %{buildroot}%{_prefix}/lib/rhc/
install -m 0755 -vd                     %{buildroot}%{_sysconfdir}/rhc/

%check
%if 0%{?fedora}
# Only go-vendor-tools are capable of validating the generated license string matches rpm's License field.
%go_vendor_license_check -c %{S:2}
%endif

%if %{with check}
# Trigger unit tests, unless explicitly disabled.
%gocheck
%endif

%clean
# Remove the module cache so rpmbuild can remove the build tree.
GOMODCACHE="%{gobuilddir}/pkg/mod" go clean -modcache
rm -rf %{buildroot}

%if 0%{?fedora}
# With go-vendor-tools, we also get a list of dependency licenses.
%global extra_files -f %{go_vendor_license_filelist}
%else
%global extra_files %{nil}
%endif

%files %{extra_files}
# Binaries
%{_bindir}/rhc-heartbeat
# Documentation
%doc README.md SECURITY.md

%changelog
%autochangelog
