package risk

func scoreSystemctl(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "stop", "restart", "disable", "kill", "poweroff", "reboot") {
		return warn("systemctl", "Risky command: controls system services.")
	}
	return Assessment{Level: Safe}
}

func scoreLaunchctl(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "unload", "bootout", "remove", "disable", "kickstart") {
		return warn("launchctl", "Risky command: controls system services.")
	}
	return Assessment{Level: Safe}
}

func scoreDocker(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	switch argv[1] {
	case "rm", "rmi":
		return warn("docker-remove", "Risky command: changes containers or infrastructure.")
	case "system":
		if len(argv) > 2 && argv[2] == "prune" {
			return warn("docker-prune", "Risky command: changes containers or infrastructure.")
		}
	case "compose":
		return scoreDockerCompose(argv[2:])
	}
	return Assessment{Level: Safe}
}

func scoreDockerCompose(args []string) Assessment {
	if len(args) == 0 {
		return Assessment{Level: Safe}
	}
	switch args[0] {
	case "down":
		if containsAny(args[1:], "-v", "--volumes") {
			return warn("docker-compose-down-volumes", "Risky command: changes containers or infrastructure.")
		}
		return warn("docker-compose-down", "Risky command: changes containers or infrastructure.")
	case "up":
		return warn("docker-compose-up", "This command writes to files or changes project state.")
	}
	return Assessment{Level: Safe}
}

func scorePodman(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	if containsAny([]string{argv[1]}, "rm", "rmi") || (argv[1] == "system" && len(argv) > 2 && argv[2] == "prune") {
		return warn("podman-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreKubectl(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "delete", "apply", "replace", "scale", "patch", "rollout") {
		return warn("kubectl-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreHelm(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "uninstall", "upgrade", "install", "rollback") {
		return warn("helm-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreTerraform(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "apply", "destroy") {
		return warn("terraform-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scorePulumi(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "up", "destroy") {
		return warn("pulumi-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}
