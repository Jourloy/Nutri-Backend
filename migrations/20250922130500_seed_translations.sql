INSERT INTO translations (namespace, translation_key, locale, value)
SELECT 'common', key, 'ru', value
FROM jsonb_each_text('{
    "meta.title": "Нутри — умный трекер питания",
    "meta.description": "Умное приложение для отслеживания питания, параметров тела и достижения целей здорового образа жизни",
    "brand.name": "Нутри",
    "tag.beta": "Бета",
    "nav.home": "Главная",
    "nav.pricing": "Тарифы",
    "auth.login": "Войти",
    "footer.description": "Умная платформа для удобного контроля за здоровьем",
    "footer.company.name": "ООО «Джоурлой»",
    "footer.company.address": "Адрес: 353217, Краснодарский край, Динской м.р-н, п. Южный, ул. Лунная, д. 14",
    "footer.company.tax": "ИНН: 2373027111",
    "footer.contacts.title": "Контакты",
    "footer.documents.title": "Документы",
    "footer.documents.privacy": "Политика конфиденциальности",
    "footer.documents.terms": "Условия использования",
    "footer.documents.offer": "Публичная оферта",
    "footer.documents.partnerCard": "Карточка контрагента",
    "footer.links.press": "Для прессы",
    "footer.links.partners": "Партнёрам",
    "footer.rights": "Все права защищены",
    "buttons.backToMain": "Вернуться на главную"
}'::jsonb);

INSERT INTO translations (namespace, translation_key, locale, value)
SELECT 'common', key, 'en', value
FROM jsonb_each_text('{
    "meta.title": "Nutri — smart nutrition tracker",
    "meta.description": "Smart app to track nutrition, body metrics and achieve your wellbeing goals",
    "brand.name": "Nutri",
    "tag.beta": "Beta",
    "nav.home": "Home",
    "nav.pricing": "Pricing",
    "auth.login": "Log in",
    "footer.description": "Smart platform that keeps your wellbeing on track",
    "footer.company.name": "Jourloy LLC",
    "footer.company.address": "Address: 14 Lunnaya str., Yuzhny, Dinskoy district, Krasnodar region, 353217",
    "footer.company.tax": "Tax ID: 2373027111",
    "footer.contacts.title": "Contacts",
    "footer.documents.title": "Documents",
    "footer.documents.privacy": "Privacy policy",
    "footer.documents.terms": "Terms of use",
    "footer.documents.offer": "Public offer",
    "footer.documents.partnerCard": "Company profile",
    "footer.links.press": "Press kit",
    "footer.links.partners": "Partners",
    "footer.rights": "All rights reserved",
    "buttons.backToMain": "Back to home"
}'::jsonb);

INSERT INTO translations (namespace, translation_key, locale, value)
SELECT 'landing', key, 'ru', value
FROM jsonb_each_text('{
    "hero.title": "Привет! Я — <brand>Нутри</brand>",
    "hero.subtitle": "Отслеживай питание, параметры тела и достигай целей, а я буду помогать тебе с этим",
    "hero.cta": "Создать профиль",
    "features.heading": "Почему нужно выбрать <brand>Нутри</brand>",
    "features.description": "Все необходимые инструменты для здорового образа жизни и достижения ваших целей",
    "features.analytics.title": "Продвинутая аналитика",
    "features.analytics.description": "Детальные графики прогресса, тренды питания и персонализированные рекомендации",
    "features.fast.title": "Быстрое добавление",
    "features.fast.description": "Интуитивный интерфейс для мгновенного ввода продуктов и автоматического расчета питательной ценности",
    "features.personal.title": "Персонализация",
    "features.personal.description": "Индивидуальные цели на основе ваших физических данных, активности и целей по весу",
    "features.motivation.title": "Система мотивации",
    "features.motivation.description": "Достижения и награды за постоянство в отслеживании питания и достижение целей",
    "pricing.title": "Базовый тариф бесплатный <highlight>НАВСЕГДА</highlight>",
    "pricing.subtitlePrimary": "В нем есть все, что нужно для отслеживания КБЖУ и этим можно начать пользоваться уже прямо сейчас!",
    "pricing.subtitleSecondary": "А платный тариф совсем недорогой и содержит в себе много полезных дополнений, которые помогут в достижении целей",
    "pricing.cta": "Узнать подробнее",
    "end.title": "Пора приступать к <highlight>учету</highlight>",
    "end.subtitle": "От результатов тебя отделяет всего лишь один клик мышкой!",
    "end.cta": "Приступить"
}'::jsonb);

INSERT INTO translations (namespace, translation_key, locale, value)
SELECT 'landing', key, 'en', value
FROM jsonb_each_text('{
    "hero.title": "Hi! I&apos;m <brand>Nutri</brand>",
    "hero.subtitle": "Track your meals, body metrics and hit your goals — I&apos;ll take care of the routine",
    "hero.cta": "Create profile",
    "features.heading": "Why choose <brand>Nutri</brand>",
    "features.description": "Everything you need to live healthier and reach your goals",
    "features.analytics.title": "Advanced analytics",
    "features.analytics.description": "Insightful charts, nutrition trends and personalised recommendations",
    "features.fast.title": "Quick logging",
    "features.fast.description": "Intuitive UI to add meals instantly with auto calorie and macro calculation",
    "features.personal.title": "Personalised",
    "features.personal.description": "Tailored goals powered by your body data, activity level and weight target",
    "features.motivation.title": "Motivation system",
    "features.motivation.description": "Collect achievements and rewards for staying on track with your plan",
    "pricing.title": "Starter plan stays free <highlight>FOREVER</highlight>",
    "pricing.subtitlePrimary": "Everything you need to track calories and macros — jump in right away",
    "pricing.subtitleSecondary": "Premium is affordable and unlocks extra guidance to keep you progressing",
    "pricing.cta": "See pricing",
    "end.title": "It''s time to start <highlight>tracking</highlight>",
    "end.subtitle": "One click stands between you and a healthier routine",
    "end.cta": "Get started"
}'::jsonb);

INSERT INTO translations (namespace, translation_key, locale, value)
SELECT 'auth', key, 'ru', value
FROM jsonb_each_text('{
    "login.title": "Вход",
    "login.username": "Имя пользователя",
    "login.password": "Пароль",
    "login.submit": "Войти",
    "login.noAccount": "У меня еще нет аккаунта",
    "login.successTitle": "Успешный вход",
    "login.successDescription": "Вы успешно вошли в систему",
    "login.errorTitle": "Ошибка",
    "login.errorDescription": "Что-то пошло не так",
    "register.title": "Регистрация",
    "register.username": "Имя пользователя",
    "register.password": "Пароль",
    "register.submit": "Зарегистрироваться",
    "register.haveAccount": "У меня уже есть аккаунт",
    "register.isAdult": "Подтверждаю, что мне уже исполнилось 18 лет",
    "register.acceptTerms": "Согласен с <terms>условиями использования</terms> и <privacy>политикой конфиденциальности</privacy>",
    "register.phoneTrap": "Телефон"
}'::jsonb);

INSERT INTO translations (namespace, translation_key, locale, value)
SELECT 'auth', key, 'en', value
FROM jsonb_each_text('{
    "login.title": "Log in",
    "login.username": "Username",
    "login.password": "Password",
    "login.submit": "Log in",
    "login.noAccount": "I don&apos;t have an account yet",
    "login.successTitle": "Signed in",
    "login.successDescription": "You are in!",
    "login.errorTitle": "Error",
    "login.errorDescription": "Something went wrong",
    "register.title": "Create account",
    "register.username": "Username",
    "register.password": "Password",
    "register.submit": "Sign up",
    "register.haveAccount": "I already have an account",
    "register.isAdult": "I confirm I am at least 18 years old",
    "register.acceptTerms": "I agree with the <terms>terms of use</terms> and <privacy>privacy policy</privacy>",
    "register.phoneTrap": "Phone"
}'::jsonb);
