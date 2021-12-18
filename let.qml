import org.kde.kirigami 2.12 as Kirigami

Item {
	item: {
		var a = 1
		a = 3
		{
			a = 2
			if (a) {
				a = 3
			}
		}
	}
}
