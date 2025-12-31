package fs

func Setup(rootfs string, root bool) error {

	if err := setupRootfs(rootfs, root); err != nil {
		return err
	}

	if err := mountProc(); err != nil {
		return err
	}

	return nil
}
