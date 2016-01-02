%define version 0.0.1

Name:		javadocr
Version:	%{version}
Release:	%{?build_number:%{build_number}}%{!?build_number:1}%{?dist}
Summary:	An automatic javadoc serving tool

Group:		Web
License:	BSD
URL:		https://github.com/lukegb/javadocr
Source0:	https://github.com/lukegb/javadocr/archive/master.tar.gz
Source1:        %{name}.service

ExclusiveArch:  %{?go_arches:%{go_arches}}%{!?go_arches:%{ix86} x86_64 %{arm}}
BuildRequires:  %{?go_compiler:compiler(go-compiler)}%{!?go_compiler:golang}
BuildRequires:	systemd
Requires(pre):	shadow-utils
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

%description
A utility for serving javadocs automatically out of a Maven repository, with
the capacity for automatically updating the served javadocs as time goes on.

%prep
%setup -q -n javadocr-master


%build

mkdir -p src/github.com/lukegb
ln -s ../../../ src/github.com/lukegb/javadocr

export GOPATH=$(pwd):%{gopath}
%gobuild -o bin/%{name} github.com/lukegb/javadocr/cmds/javadocr


%install
install -D -p -m 0755 bin/%{name} %{buildroot}%{_bindir}/%{name}
install -D -p -m 0644 %{SOURCE1} %{buildroot}%{_unitdir}/%{name}.service

%pre
getent group %{name} >/dev/null || groupadd -r %{name}
getent passwd %{name} >/dev/null || useradd -r -g %{name} -d %{_sharedstatedir}/%{name} \
	-s /sbin/nologin -c "%{name} user" %{name}

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun %{name}.service


%files
%{_bindir}/%{name}
%{_unitdir}/%{name}.service


%changelog
* Fri Jan 1 2016 Luke Granger-Brown <git@lukegb.com> - 0.0.1
- initial package
