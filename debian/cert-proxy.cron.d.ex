#
# Regular cron jobs for the cert-proxy package
#
0 4	* * *	root	[ -x /usr/bin/cert-proxy_maintenance ] && /usr/bin/cert-proxy_maintenance
