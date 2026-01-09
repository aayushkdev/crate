package fs

func Setup(rootfs string, rootless bool) error {

	hostFDs, err := OpenHostDevFDs(rootless)
	if err != nil {
		return err
	}

	if err := setupRootfs(rootfs, rootless); err != nil {
		return err
	}

	if err := mountProc(); err != nil {
		return err
	}

	if err := mountDev(rootless, hostFDs); err != nil {
		return err
	}

	CloseHostFDs(hostFDs)

	return nil
}
