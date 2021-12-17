import org.kde.kirigami 2.10 as Kirigami

Hello {
	property string mald: 1
	mald: {
		with (hello) {
			mald = 2
		}
	}
}

