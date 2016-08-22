/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
*/

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>
#include <pwd.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "config_parser.h"
#include "sessiondir.h"
#include "rootfs/rootfs.h"


int singularity_mount_home(void) {
    char *tmpdirpath;
    char *homedir;
    char *homedir_source;
    char *homedir_base = NULL;
    char *container_dir = singularity_rootfs_dir();
    char *sessiondir = singularity_sessiondir_get();
    struct passwd *pw;
    uid_t uid = priv_getuid();

    config_rewind();
    if ( config_get_key_bool("mount home", 1) <= 0 ) {
        message(VERBOSE, "Skipping tmp dir mounting (per config)\n");
        return(0);
    }

    errno = 0;
    if ( ( pw = getpwuid(uid) ) == NULL ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
            message(VERBOSE3, "Not mounting home directory as passwd entry for %d not found.\n", uid);
            return(1);
        } else {
            message(ERROR, "Failed to lookup username for UID %d: %s\n", getuid, strerror(errno));
            ABORT(255);
        }
    }

    message(DEBUG, "Obtaining user's homedir\n");
    homedir = pw->pw_dir;

    // Figure out home directory source
    if ( ( homedir_source = getenv("SINGULARITY_HOME") ) != NULL ) {
        config_rewind();
        if ( config_get_key_bool("user bind control", 1) <= 0 ) {
            message(ERROR, "User bind control is disabled by system administrator\n");
            ABORT(5);
        }

        message(VERBOSE2, "Set the home directory source (via envar) to: %s\n", homedir_source);
    } else if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
        if ( ( tmpdirpath = getenv("SINGULARITY_WORKDIR") ) != NULL ) {
            config_rewind();
            if ( config_get_key_bool("user bind control", 1) <= 0 ) {
                message(ERROR, "User bind control is disabled by system administrator\n");
                ABORT(5);
            }

            homedir_source = joinpath(tmpdirpath, "/home");
        } else {
            // TODO: Randomize tmp_home, so multiple calls to the same container don't overlap
            homedir_source = joinpath(sessiondir, "/home.tmp");
        }
        if ( s_mkpath(homedir_source, 0755) < 0 ) {
            message(ERROR, "Could not create temporary home directory %s: %s\n", homedir_source, strerror(errno));
            ABORT(255);
        } else {
            message(VERBOSE2, "Set the contained home directory source to: %s\n", homedir_source);
        }

    } else if ( is_dir(homedir) == 0 ) {
        homedir_source = strdup(homedir);
        message(VERBOSE2, "Set base the home directory source to: %s\n", homedir_source);
    } else {
        message(ERROR, "Could not identify home directory path: %s\n", homedir_source);
        ABORT(255);
    }

    // Create a location to stage the directories
    if ( s_mkpath(homedir_source, 0755) < 0 ) {
        message(ERROR, "Failed creating home directory bind path\n");
    }

    // Create a location to stage the directories
    if ( s_mkpath(joinpath(sessiondir, homedir), 0755) < 0 ) {
        message(ERROR, "Failed creating home directory bind path\n");
    }

    // Check to make sure whatever we were given as the home directory is really ours
    message(DEBUG, "Checking permissions on home directory: %s\n", homedir_source);
    if ( is_owner(homedir_source, uid) < 0 ) {
        message(ERROR, "Home directory permissions incorrect: %s\n", homedir_source);
        ABORT(255);
    }

    // Figure out where we should mount the home directory in the container
    message(DEBUG, "Trying to create home dir within container\n");
    if ( singularity_rootfs_overlay_enabled() > 0 ) {
        priv_escalate();
        if ( s_mkpath(joinpath(container_dir, homedir), 0750) == 0 ) {
            priv_drop();
            message(DEBUG, "Created home directory within the container: %s\n", homedir);
            homedir_base = strdup(homedir);
        } else {
            priv_drop();
        }
    }

    if ( homedir_base == NULL ) {
        if ( ( homedir_base = container_basedir(container_dir, homedir) ) != NULL ) {
            message(DEBUG, "Could not create directory within container, set base bind point to: %s\n", homedir_base);
        } else {
            message(ERROR, "No bind point available for home directory: %s\n", homedir);
            ABORT(255);
        }
    }

    priv_escalate();
    // First mount the real home directory to the stage
    message(VERBOSE, "Mounting home directory to stage: %s->%s\n", homedir_source, joinpath(sessiondir, homedir));
    if ( mount(homedir_source, joinpath(sessiondir, homedir), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        message(ERROR, "Failed to mount home directory to stage: %s\n", strerror(errno));
        ABORT(255);
    }
    // Then mount the stage to the container
    message(VERBOSE, "Mounting staged home directory into container: %s->%s\n", joinpath(sessiondir, homedir_base), joinpath(container_dir, homedir_base));
    if ( mount(joinpath(sessiondir, homedir_base), joinpath(container_dir, homedir_base), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        message(ERROR, "Failed to mount staged home directory into container: %s\n", strerror(errno));
        ABORT(255);
    }
    priv_drop();

    return(0);
}
