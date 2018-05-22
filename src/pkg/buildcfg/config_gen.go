// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildcfg

//go:generate go run confgen/gen.go "example.h"

var (
	// PREFIX install dir prefix
	PREFIX     = "/usr/local"
	EXECPREFIX = PREFIX
	// BINDIR singularity bin path
	BINDIR      = EXECPREFIX + "/bin"
	DATAROOTDIR = PREFIX + "/share"
	DATADIR     = DATAROOTDIR

	// SBINDIR path for singularity helper bins
	SBINDIR = "/usr/local/libexec/singularity/bin"
	// LIBEXECDIR path for libexec folder
	LIBEXECDIR = "/usr/local/libexec"

	PACKAGE_NAME      = "singularity"
	PACKAGE_TARNAME   = "singularity"
	PACKAGE_VERSION   = "3.0"
	PACKAGE_STRING    = "singularity + 3.0"
	PACKAGE_BUGREPORT = "gmkurtzer@gmail.com"
	PACKAGE_URL       = ""
	BUILDDIR          = "builddir"

	SYSCONFDIR                = PREFIX + "/etc"
	SHAREDSTARTEDIR           = PREFIX + "/com"
	LOCALSTATEDIR             = PREFIX + "/var"
	INCLUDEDIR                = PREFIX + "/include"
	OLDINCLUDEDIR             = "/usr/include"
	DOCDIR                    = DATAROOTDIR + "/doc/" + PACKAGE_TARNAME
	INFODIR                   = DATAROOTDIR + "/info"
	HTMLDIR                   = DOCDIR
	DVIDIR                    = DOCDIR
	PDFDIR                    = DOCDIR
	PSDIR                     = DOCDIR
	LIBDIR                    = EXECPREFIX + "/lib"
	LOCALEDIR                 = DATAROOTDIR + "/locale"
	MANDIR                    = DATAROOTDIR + "/man"
	CONTAINER_MOUNTDIR        = LOCALSTATEDIR + "/singularity/mnt/container"
	CONTAINER_FINALDIR        = LOCALSTATEDIR + "/singularity/mnt/final"
	CONTAINER_OVERLAY         = LOCALSTATEDIR + "/singularity/mnt/overlay"
	SESSIONDIR                = LOCALSTATEDIR + "/singularity/mnt/session"
	NS_CLONE_NEWPID           = 1
	NS_CLONE_FS               = 1
	NS_CLONE_NEWNS            = 1
	NS_CLONE_NEWUSER          = 1
	NS_CLONE_NEWIPC           = 1
	NS_CLONE_NEWNET           = 1
	NS_CLONE_NEWUTS           = 1
	SINGULARITY_NO_NEW_PRIVS  = 1
	SINGULARITY_MS_SLAVE      = 1
	USER_CAPABILITIES         = 1
	SINGULARITY_SECUREBITS    = 1
	SINGULARITY_NO_SETNS      = 1
	SINGULARITY_SETNS_SYSCALL = 1
)
