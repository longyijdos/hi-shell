package risk

func scoreSystemPackageManager(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	switch argv[1] {
	case "remove", "purge", "autoremove":
		return warn("package-remove", "Risky command: removes installed packages.")
	case "install", "update", "upgrade":
		return warn("package-install", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreBrew(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "install", "uninstall", "upgrade", "update") {
		return warn("brew-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreNodePackageManager(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	if argv[0] == "npm" && argv[1] == "run" {
		if len(argv) > 2 && containsAny([]string{argv[2]}, "dev", "start") {
			return warn("npm-run", "This command starts or changes local project state.")
		}
		return Assessment{Level: Safe}
	}
	if containsAny([]string{argv[1]}, "install", "i", "add", "update", "uninstall", "remove") {
		return warn("node-package-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scorePip(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "install", "uninstall") {
		return warn("pip-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreGo(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "get", "install", "run") {
		return warn("go-change", "This command writes to files or changes project state.")
	}
	return Assessment{Level: Safe}
}

func scoreCargo(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "add", "install", "update") {
		return warn("cargo-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreBundle(argv []string) Assessment {
	if len(argv) > 1 && argv[1] == "install" {
		return warn("bundle-install", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}
